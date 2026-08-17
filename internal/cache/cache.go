package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Meta stores cached image metadata in a JSON sidecar file.
type Meta struct {
	ContentType   string    `json:"content_type"`
	ContentLength int64     `json:"content_length"`
	CreatedAt     time.Time `json:"created_at"`
	LastAccessed  time.Time `json:"last_accessed"`
}

// DiskCache implements a two-level sharded disk cache for images with TTL and LRU eviction.
type DiskCache struct {
	dir      string
	ttl      time.Duration
	maxBytes int64
	sf       singleflight.Group
	mu       sync.RWMutex
}

// NewDiskCache initializes a DiskCache at dir.
func NewDiskCache(dir string, ttl time.Duration, maxBytes int64) (*DiskCache, error) {
	imagesDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache images directory: %w", err)
	}

	return &DiskCache{
		dir:      dir,
		ttl:      ttl,
		maxBytes: maxBytes,
	}, nil
}

// Key computes the SHA-256 hex digest of urlStr.
func (c *DiskCache) Key(urlStr string) string {
	sum := sha256.Sum256([]byte(urlStr))
	return hex.EncodeToString(sum[:])
}

// paths returns the shard directory, image file path, and json metadata file path for a hash.
func (c *DiskCache) paths(hash string) (shardDir, imgPath, jsonPath string) {
	if len(hash) < 4 {
		shardDir = filepath.Join(c.dir, "images", "00", "00")
	} else {
		shardDir = filepath.Join(c.dir, "images", hash[:2], hash[2:4])
	}
	imgPath = filepath.Join(shardDir, hash+".img")
	jsonPath = filepath.Join(shardDir, hash+".json")
	return shardDir, imgPath, jsonPath
}

// GetPath returns the cached image path and metadata if present and unexpired.
// Updates LastAccessed timestamp on cache hit.
func (c *DiskCache) GetPath(urlStr string) (string, Meta, bool) {
	hash := c.Key(urlStr)
	_, imgPath, jsonPath := c.paths(hash)

	c.mu.RLock()
	imgInfo, err := os.Stat(imgPath)
	if err != nil || imgInfo.IsDir() {
		c.mu.RUnlock()
		return "", Meta{}, false
	}

	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		c.mu.RUnlock()
		return "", Meta{}, false
	}

	var meta Meta
	if err := json.Unmarshal(jsonBytes, &meta); err != nil {
		c.mu.RUnlock()
		return "", Meta{}, false
	}

	if c.ttl > 0 && !meta.CreatedAt.IsZero() && time.Since(meta.CreatedAt) > c.ttl {
		c.mu.RUnlock()
		return "", Meta{}, false
	}
	c.mu.RUnlock()

	meta.LastAccessed = time.Now()
	if updatedBytes, err := json.Marshal(meta); err == nil {
		c.mu.Lock()
		_ = os.WriteFile(jsonPath, updatedBytes, 0644)
		c.mu.Unlock()
	}

	return imgPath, meta, true
}

type fetchResult struct {
	imgPath string
	meta    Meta
}

// GetOrFetch retrieves an image from cache or calls fetcher to populate the cache.
// Deduplicates concurrent requests for the same URL using singleflight.
func (c *DiskCache) GetOrFetch(
	ctx context.Context,
	urlStr string,
	fetcher func(ctx context.Context) (io.ReadCloser, Meta, error),
) (io.ReadCloser, Meta, error) {
	if imgPath, meta, ok := c.GetPath(urlStr); ok {
		f, err := os.Open(imgPath)
		if err == nil {
			return f, meta, nil
		}
	}

	hash := c.Key(urlStr)
	res, err, _ := c.sf.Do(hash, func() (any, error) {
		if imgPath, meta, ok := c.GetPath(urlStr); ok {
			return fetchResult{imgPath: imgPath, meta: meta}, nil
		}

		shardDir, imgPath, jsonPath := c.paths(hash)
		if err := os.MkdirAll(shardDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create shard directory: %w", err)
		}

		rc, meta, err := fetcher(ctx)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		tmpFile, err := os.CreateTemp(shardDir, hash+".tmp.*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		cleanupTmp := true
		defer func() {
			if cleanupTmp {
				_ = tmpFile.Close()
				_ = os.Remove(tmpPath)
			}
		}()

		written, err := io.Copy(tmpFile, rc)
		if err != nil {
			return nil, fmt.Errorf("failed to write image stream to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return nil, fmt.Errorf("failed to close temp file: %w", err)
		}

		if meta.CreatedAt.IsZero() {
			meta.CreatedAt = time.Now()
		}
		meta.LastAccessed = time.Now()
		if meta.ContentLength <= 0 {
			meta.ContentLength = written
		}

		metaBytes, err := json.Marshal(meta)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		c.mu.Lock()
		defer c.mu.Unlock()

		if err := os.Rename(tmpPath, imgPath); err != nil {
			return nil, fmt.Errorf("failed to rename temp file to image: %w", err)
		}

		if err := os.WriteFile(jsonPath, metaBytes, 0644); err != nil {
			_ = os.Remove(imgPath)
			return nil, fmt.Errorf("failed to write json metadata: %w", err)
		}
		cleanupTmp = false

		return fetchResult{imgPath: imgPath, meta: meta}, nil
	})

	if err != nil {
		return nil, Meta{}, err
	}

	fr := res.(fetchResult)
	f, err := os.Open(fr.imgPath)
	if err != nil {
		return nil, Meta{}, fmt.Errorf("failed to open cached image: %w", err)
	}

	return f, fr.meta, nil
}

// CopyTo copies or links a cached image to destPath.
// If the image is not in cache, fetcher is invoked.
// Attempts os.Link first, falling back to byte copy on link failure.
func (c *DiskCache) CopyTo(
	ctx context.Context,
	urlStr string,
	destPath string,
	fetcher func(ctx context.Context) (io.ReadCloser, Meta, error),
) error {
	var cachedImagePath string
	if imgPath, _, ok := c.GetPath(urlStr); ok {
		cachedImagePath = imgPath
	} else {
		rc, _, err := c.GetOrFetch(ctx, urlStr, fetcher)
		if err != nil {
			return err
		}
		_ = rc.Close()

		hash := c.Key(urlStr)
		_, imgPath, _ := c.paths(hash)
		cachedImagePath = imgPath
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	_ = os.Remove(destPath)

	if err := os.Link(cachedImagePath, destPath); err == nil {
		return nil
	}

	srcFile, err := os.Open(cachedImagePath)
	if err != nil {
		return fmt.Errorf("failed to open cached image for copy: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy cached image to destination: %w", err)
	}

	return nil
}

type cacheEntry struct {
	imgPath      string
	jsonPath     string
	size         int64
	createdAt    time.Time
	lastAccessed time.Time
}

// workerCount returns a bounded number of worker goroutines suitable for taskCount.
func workerCount(taskCount int) int {
	workers := runtime.NumCPU() * 2
	if workers < 4 {
		workers = 4
	}
	if workers > 16 {
		workers = 16
	}
	if taskCount > 0 && workers > taskCount {
		workers = taskCount
	}
	return workers
}

// PurgeExpiredAndLRU scans the cache images directory, removes expired files based on ttl,
// and if remaining total size > maxBytes, removes oldest accessed files until total size <= maxBytes.
// Uses concurrent worker pools to scan shards and delete files without holding a global write lock.
// Returns total bytes purged.
func (c *DiskCache) PurgeExpiredAndLRU() (int64, error) {
	start := time.Now()
	imagesDir := filepath.Join(c.dir, "images")
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var (
		shardDirs   []string
		directFiles []fs.DirEntry
		now         = time.Now()
	)

	for _, entry := range entries {
		if entry.IsDir() {
			shardDirs = append(shardDirs, filepath.Join(imagesDir, entry.Name()))
		} else {
			directFiles = append(directFiles, entry)
		}
	}

	if len(shardDirs) == 0 && len(directFiles) == 0 {
		return 0, nil
	}

	var (
		resultsMu   sync.Mutex
		activeItems []cacheEntry
		totalPurged int64
	)

	processFile := func(path string, d fs.DirEntry) (entry *cacheEntry, purged int64) {
		if strings.HasSuffix(path, ".img") {
			jsonPath := strings.TrimSuffix(path, ".img") + ".json"
			info, err := d.Info()
			if err != nil {
				return nil, 0
			}

			size := info.Size()
			var meta Meta
			jsonBytes, err := os.ReadFile(jsonPath)
			if err == nil {
				_ = json.Unmarshal(jsonBytes, &meta)
			}
			if meta.CreatedAt.IsZero() {
				meta.CreatedAt = info.ModTime()
			}
			if meta.LastAccessed.IsZero() {
				meta.LastAccessed = meta.CreatedAt
			}

			if c.ttl > 0 && now.Sub(meta.CreatedAt) > c.ttl {
				_ = os.Remove(path)
				_ = os.Remove(jsonPath)
				return nil, size
			}

			return &cacheEntry{
				imgPath:      path,
				jsonPath:     jsonPath,
				size:         size,
				createdAt:    meta.CreatedAt,
				lastAccessed: meta.LastAccessed,
			}, 0
		} else if strings.HasSuffix(path, ".json") {
			imgPath := strings.TrimSuffix(path, ".json") + ".img"
			if _, err := os.Stat(imgPath); os.IsNotExist(err) {
				if info, err := d.Info(); err == nil && now.Sub(info.ModTime()) > 5*time.Second {
					_ = os.Remove(path)
				}
			}
		} else if strings.HasSuffix(path, ".tmp") || strings.Contains(d.Name(), ".tmp.") {
			if info, err := d.Info(); err == nil && now.Sub(info.ModTime()) > 30*time.Minute {
				_ = os.Remove(path)
			}
		}
		return nil, 0
	}

	// Process any stray direct files in imagesDir
	for _, d := range directFiles {
		path := filepath.Join(imagesDir, d.Name())
		if entry, purged := processFile(path, d); entry != nil {
			activeItems = append(activeItems, *entry)
		} else if purged > 0 {
			totalPurged += purged
		}
	}

	// Scan shard directories in parallel using a bounded worker pool
	if len(shardDirs) > 0 {
		numWorkers := workerCount(len(shardDirs))
		shardChan := make(chan string, len(shardDirs))
		for _, dir := range shardDirs {
			shardChan <- dir
		}
		close(shardChan)

		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var localEntries []cacheEntry
				var localPurged int64

				for dirPath := range shardChan {
					_ = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, walkErr error) error {
						if walkErr != nil || d == nil {
							return nil
						}
						if d.IsDir() {
							return nil
						}

						if entry, purged := processFile(path, d); entry != nil {
							localEntries = append(localEntries, *entry)
						} else if purged > 0 {
							localPurged += purged
						}
						return nil
					})
				}

				if len(localEntries) > 0 || localPurged > 0 {
					resultsMu.Lock()
					activeItems = append(activeItems, localEntries...)
					totalPurged += localPurged
					resultsMu.Unlock()
				}
			}()
		}
		wg.Wait()
	}

	// LRU Pruning if totalSize > maxBytes
	var totalSize int64
	for _, item := range activeItems {
		totalSize += item.size
	}

	if c.maxBytes > 0 && totalSize > c.maxBytes {
		sort.Slice(activeItems, func(i, j int) bool {
			return activeItems[i].lastAccessed.Before(activeItems[j].lastAccessed)
		})

		var toDelete []cacheEntry
		for _, item := range activeItems {
			if totalSize <= c.maxBytes {
				break
			}
			toDelete = append(toDelete, item)
			totalSize -= item.size
			totalPurged += item.size
		}

		if len(toDelete) > 0 {
			delWorkers := workerCount(len(toDelete))
			delChan := make(chan cacheEntry, len(toDelete))
			for _, item := range toDelete {
				delChan <- item
			}
			close(delChan)

			var delWg sync.WaitGroup
			for i := 0; i < delWorkers; i++ {
				delWg.Add(1)
				go func() {
					defer delWg.Done()
					for item := range delChan {
						_ = os.Remove(item.imgPath)
						_ = os.Remove(item.jsonPath)
					}
				}()
			}
			delWg.Wait()
		}
	}

	if totalPurged > 0 {
		slog.Info("cache: purge completed",
			slog.Int64("purged_bytes", totalPurged),
			slog.Duration("elapsed", time.Since(start)),
		)
	} else {
		slog.Debug("cache: purge completed",
			slog.Int64("purged_bytes", 0),
			slog.Duration("elapsed", time.Since(start)),
		)
	}

	return totalPurged, nil
}

// StartCleanupWorker periodically runs PurgeExpiredAndLRU in a background goroutine.
func (c *DiskCache) StartCleanupWorker(ctx context.Context, interval time.Duration) {
	slog.Info("cache: background cleanup worker started", slog.Duration("interval", interval))
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := c.PurgeExpiredAndLRU(); err != nil {
					slog.Error("cache: cleanup worker purge failed", slog.String("error", err.Error()))
				}
			}
		}
	}()
}

// Stats returns the total size in bytes and count of cached .img files in the images directory.
func (c *DiskCache) Stats() (int64, int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	imagesDir := filepath.Join(c.dir, "images")
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var totalSize int64
	var count int

	err := filepath.WalkDir(imagesDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".img") {
			info, err := d.Info()
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			totalSize += info.Size()
			count++
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("failed to calculate cache stats: %w", err)
	}

	return totalSize, count, nil
}

// Clear removes all files in the cache images directory and recreates the empty images directory.
func (c *DiskCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	imagesDir := filepath.Join(c.dir, "images")
	if err := os.RemoveAll(imagesDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear cache directory: %w", err)
	}

	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate cache directory: %w", err)
	}

	return nil
}

