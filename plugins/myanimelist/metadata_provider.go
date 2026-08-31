package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	sdk "github.com/tubruk/kiyomi/plugin-sdk"
)

const malFields = "id,title,main_picture,alternative_titles,start_date,end_date,synopsis,mean,rank,popularity,num_list_users,num_scoring_users,nsfw,created_at,updated_at,media_type,status,genres,num_volumes,num_chapters,authors{first_name,last_name},pictures,background"

func parseAliases(node malMangaNode) []string {
	var aliases []string
	seen := make(map[string]bool)
	seen[node.Title] = true

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			aliases = append(aliases, s)
		}
	}

	add(node.AlternativeTitles.En)
	add(node.AlternativeTitles.Ja)
	for _, syn := range node.AlternativeTitles.Synonyms {
		add(syn)
	}
	return aliases
}

func parseAuthors(authorsList []malAuthorNode) (string, string) {
	var authors []string
	var artists []string
	seenAuthor := make(map[string]bool)
	seenArtist := make(map[string]bool)

	for _, a := range authorsList {
		firstName := strings.TrimSpace(a.Node.FirstName)
		lastName := strings.TrimSpace(a.Node.LastName)
		name := strings.TrimSpace(fmt.Sprintf("%s %s", firstName, lastName))
		if name == "" {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(a.Role))
		if strings.Contains(role, "art") {
			if !seenArtist[name] {
				seenArtist[name] = true
				artists = append(artists, name)
			}
		}
		if strings.Contains(role, "story") || strings.Contains(role, "author") || (!strings.Contains(role, "art") && role != "") || role == "" {
			if !seenAuthor[name] {
				seenAuthor[name] = true
				authors = append(authors, name)
			}
		}
	}
	return strings.Join(authors, ", "), strings.Join(artists, ", ")
}

// Search queries MyAnimeList for manga matching query or browsing mode.
func (p *MyAnimeListPlugin) Search(ctx context.Context, query string, opts sdk.SearchOptions) ([]sdk.SearchResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := opts.Offset

	query = strings.TrimSpace(query)

	var endpoint string
	if query != "" {
		endpoint = fmt.Sprintf("%s/manga?q=%s&limit=%d&offset=%d&fields=%s",
			p.getBaseURL(), url.QueryEscape(query), limit, offset, url.QueryEscape("id,title,main_picture,alternative_titles,synopsis,mean,status"))
	} else if opts.Mode == "latest" {
		// MyAnimeList ranking: all, manga, novels, oneshots, doujin, manhwa, manhua, popularity, favorite
		// For latest or general browsing when query is empty, use ranking manga or all
		endpoint = fmt.Sprintf("%s/manga/ranking?ranking_type=manga&limit=%d&offset=%d&fields=%s",
			p.getBaseURL(), limit, offset, url.QueryEscape("id,title,main_picture,alternative_titles,synopsis,mean,status"))
	} else {
		// Default to popular (by popularity ranking)
		endpoint = fmt.Sprintf("%s/manga/ranking?ranking_type=bypopularity&limit=%d&offset=%d&fields=%s",
			p.getBaseURL(), limit, offset, url.QueryEscape("id,title,main_picture,alternative_titles,synopsis,mean,status"))
	}

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, fmt.Errorf("myanimelist search: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("myanimelist search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("myanimelist search status %d", resp.StatusCode)
	}

	var apiResp malSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("myanimelist search decode: %w", err)
	}

	var results []sdk.SearchResult
	for _, item := range apiResp.Data {
		node := item.Node
		remoteID := strconv.Itoa(node.ID)

		coverURL := node.MainPicture.Large
		if coverURL == "" {
			coverURL = node.MainPicture.Medium
		}

		results = append(results, sdk.SearchResult{
			RemoteID:     remoteID,
			Title:        node.Title,
			Aliases:      parseAliases(node),
			CoverURL:     coverURL,
			URL:          fmt.Sprintf("https://myanimelist.net/manga/%d", node.ID),
			Availability: sdk.AvailabilityAvailable,
		})
	}

	return results, nil
}

// Details fetches manga metadata by remote ID from MyAnimeList.
func (p *MyAnimeListPlugin) Details(ctx context.Context, remoteID string) (sdk.MangaMetadata, error) {
	remoteID = strings.TrimSpace(remoteID)
	endpoint := fmt.Sprintf("%s/manga/%s?fields=%s", p.getBaseURL(), remoteID, url.QueryEscape(malFields))

	req, err := p.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("myanimelist details: %w", err)
	}

	resp, err := p.doRequest(req)
	if err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("myanimelist details request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return sdk.MangaMetadata{}, fmt.Errorf("myanimelist details status %d", resp.StatusCode)
	}

	var node malMangaNode
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return sdk.MangaMetadata{}, fmt.Errorf("myanimelist details decode: %w", err)
	}

	authorStr, artistStr := parseAuthors(node.Authors)

	var genres []string
	for _, g := range node.Genres {
		if g.Name != "" {
			genres = append(genres, g.Name)
		}
	}

	readingMode := sdk.ReadingModeRTL
	mediaType := strings.ToLower(strings.TrimSpace(node.MediaType))
	if mediaType == "manhwa" || mediaType == "manhua" {
		readingMode = sdk.ReadingModeLongstrip
	}

	coverURL := node.MainPicture.Large
	if coverURL == "" {
		coverURL = node.MainPicture.Medium
	}

	return sdk.MangaMetadata{
		RemoteID:      strconv.Itoa(node.ID),
		Title:         node.Title,
		Aliases:       parseAliases(node),
		CoverURL:      coverURL,
		Synopsis:      node.Synopsis,
		Status:        node.Status,
		Author:        authorStr,
		Artist:        artistStr,
		Genres:        genres,
		TotalChapters: node.NumChapters,
		ReadingMode:   readingMode,
		Score:         node.Mean,
		URL:           fmt.Sprintf("https://myanimelist.net/manga/%d", node.ID),
		Availability:  sdk.AvailabilityAvailable,
	}, nil
}

// Cover returns cover image reference for the specified manga.
func (p *MyAnimeListPlugin) Cover(ctx context.Context, remoteID string, size sdk.ImageSize) (sdk.ImageRef, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return sdk.ImageRef{}, err
	}
	return sdk.ImageRef{URL: details.CoverURL}, nil
}

// Aliases returns alternative titles for the specified manga.
func (p *MyAnimeListPlugin) Aliases(ctx context.Context, remoteID string) ([]string, error) {
	details, err := p.Details(ctx, remoteID)
	if err != nil {
		return nil, err
	}
	return details.Aliases, nil
}
