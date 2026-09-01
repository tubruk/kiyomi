package main

// malSearchResponse represents the payload returned by GET /v2/manga?q=... or GET /v2/manga/ranking?ranking_type=...
type malSearchResponse struct {
	Data []struct {
		Node malMangaNode `json:"node"`
	} `json:"data"`
	Paging struct {
		Next     string `json:"next"`
		Previous string `json:"previous"`
	} `json:"paging"`
}

// malPicture represents picture URLs from MAL.
type malPicture struct {
	Medium string `json:"medium"`
	Large  string `json:"large"`
}

// malGenre represents a genre object from MAL.
type malGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// malAuthorNode represents an author or artist node.
type malAuthorNode struct {
	Node struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"node"`
	Role string `json:"role"`
}

// malAlternativeTitles represents alternative titles for a manga.
type malAlternativeTitles struct {
	Synonyms []string `json:"synonyms"`
	En       string   `json:"en"`
	Ja       string   `json:"ja"`
}

// malMangaNode represents the manga item in MAL search or details response.
type malMangaNode struct {
	ID                int                  `json:"id"`
	Title             string               `json:"title"`
	MainPicture       malPicture           `json:"main_picture"`
	AlternativeTitles malAlternativeTitles `json:"alternative_titles"`
	StartDate         string               `json:"start_date"`
	EndDate           string               `json:"end_date"`
	Synopsis          string               `json:"synopsis"`
	Mean              float32              `json:"mean"`
	Rank              int                  `json:"rank"`
	Popularity        int                  `json:"popularity"`
	NumListUsers      int                  `json:"num_list_users"`
	NumScoringUsers   int                  `json:"num_scoring_users"`
	NSFW              string               `json:"nsfw"`
	CreatedAt         string               `json:"created_at"`
	UpdatedAt         string               `json:"updated_at"`
	MediaType         string               `json:"media_type"`
	Status            string               `json:"status"`
	Genres            []malGenre           `json:"genres"`
	NumVolumes        int                  `json:"num_volumes"`
	NumChapters       int                  `json:"num_chapters"`
	Authors           []malAuthorNode      `json:"authors"`
	Serialization     []struct {
		Node struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"node"`
		Role string `json:"role"`
	} `json:"serialization"`
	Pictures          []malPicture         `json:"pictures"`
	Background        string               `json:"background"`
}
