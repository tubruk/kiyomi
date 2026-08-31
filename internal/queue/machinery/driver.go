package machinery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/RichardKnop/machinery/v2"
	redisBackend "github.com/RichardKnop/machinery/v2/backends/redis"
	redisBroker "github.com/RichardKnop/machinery/v2/brokers/redis"
	machineryConfig "github.com/RichardKnop/machinery/v2/config"
	eagerLock "github.com/RichardKnop/machinery/v2/locks/eager"
	"github.com/RichardKnop/machinery/v2/tasks"

	"github.com/tubruk/kiyomi/internal/queue"
)

// Driver implements queue.Enqueuer and queue.Worker interfaces using Machinery.
type Driver struct {
	server *machinery.Server
	worker *machinery.Worker
	tag    string
}

func parseRedisURI(uri string) ([]string, int, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, 0, err
	}
	host := u.Host
	if host == "" {
		host = "localhost:6379"
	}
	if u.User != nil {
		password, hasPassword := u.User.Password()
		if hasPassword {
			username := u.User.Username()
			if username != "" {
				host = username + ":" + password + "@" + host
			} else {
				host = password + "@" + host
			}
		}
	}
	db := 0
	if u.Path != "" {
		trimmed := strings.TrimPrefix(u.Path, "/")
		if val, err := strconv.Atoi(trimmed); err == nil {
			db = val
		}
	}
	return []string{host}, db, nil
}

// NewDriver initializes a new Machinery queue driver wrapper.
func NewDriver(cfg *machineryConfig.Config, tag string) (*Driver, error) {
	brokerAddrs, brokerDB, err := parseRedisURI(cfg.Broker)
	if err != nil {
		return nil, fmt.Errorf("parse broker URI: %w", err)
	}
	backendAddrs, backendDB, err := parseRedisURI(cfg.ResultBackend)
	if err != nil {
		return nil, fmt.Errorf("parse result backend URI: %w", err)
	}

	broker := redisBroker.NewGR(cfg, brokerAddrs, brokerDB)
	backend := redisBackend.NewGR(cfg, backendAddrs, backendDB)
	lock := eagerLock.New()

	server := machinery.NewServer(cfg, broker, backend, lock)
	return &Driver{
		server: server,
		tag:    tag,
	}, nil
}

// Enqueue sends a job task to the Machinery queue.
func (d *Driver) Enqueue(ctx context.Context, job *queue.Job) error {
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	signature := &tasks.Signature{
		UUID: job.ID,
		Name: job.Type,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(jobJSON),
			},
		},
	}

	if job.MaxRetries > 0 {
		signature.RetryCount = job.MaxRetries
	}

	_, err = d.server.SendTaskWithContext(ctx, signature)
	if err != nil {
		return fmt.Errorf("send machinery task: %w", err)
	}
	return nil
}

// Register registers a new task handler with Machinery.
func (d *Driver) Register(jobType string, handler queue.JobHandler) error {
	err := d.server.RegisterTask(jobType, func(jobJSON string) error {
		var job queue.Job
		if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
			return fmt.Errorf("unmarshal job in machinery worker: %w", err)
		}
		return handler.Handle(context.Background(), &job)
	})
	if err != nil {
		return fmt.Errorf("register machinery task: %w", err)
	}
	return nil
}

// SetGroupLimit is a no-op for the Machinery driver. Machinery manages its own
// concurrency via worker pool sizing rather than per-group limits. The method
// exists to satisfy queue.Worker.
func (d *Driver) SetGroupLimit(_ string, _ int) {}

// Start launches the Machinery worker in a background goroutine.
func (d *Driver) Start(ctx context.Context) error {
	worker := d.server.NewWorker(d.tag, 0)
	d.worker = worker

	go func() {
		// Launch worker (blocking call)
		_ = worker.Launch()
	}()

	return nil
}

// Stop halts the Machinery worker.
func (d *Driver) Stop(ctx context.Context) error {
	if d.worker != nil {
		d.worker.Quit()
	}
	return nil
}
