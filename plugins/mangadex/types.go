package main

// mangaDexSearchResponse represents the JSON response returned by the MangaDex /manga search endpoint.
type mangaDexSearchResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title                        map[string]string `json:"title"`
			AvailableTranslatedLanguages []string          `json:"availableTranslatedLanguages"`
			LatestUploadedChapter        *string           `json:"latestUploadedChapter"`
		} `json:"attributes"`
		Relationships []struct {
			Type       string `json:"type"`
			Attributes struct {
				FileName string `json:"fileName"`
			} `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}

// mangaDexDetailsResponse represents the JSON response returned by the MangaDex /manga/{id} endpoint.
type mangaDexDetailsResponse struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Title                        map[string]string `json:"title"`
			Description                  map[string]string `json:"description"`
			Status                       string            `json:"status"`
			OriginalLanguage             string            `json:"originalLanguage"`
			AvailableTranslatedLanguages []string          `json:"availableTranslatedLanguages"`
			LatestUploadedChapter        *string           `json:"latestUploadedChapter"`
			Tags                         []struct {
				Attributes struct {
					Name map[string]string `json:"name"`
				} `json:"attributes"`
			} `json:"tags"`
		} `json:"attributes"`
		Relationships []struct {
			Type       string `json:"type"`
			Attributes struct {
				FileName string `json:"fileName"`
				Name     string `json:"name"`
			} `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}

// mangaDexFeedResponse represents the JSON response returned by the MangaDex /manga/{id}/feed endpoint.
type mangaFeedResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Volume    string `json:"volume"`
			Chapter   string `json:"chapter"`
			Title     string `json:"title"`
			PublishAt string `json:"publishAt"`
		} `json:"attributes"`
	} `json:"data"`
}

// mangaDexAtHomeResponse represents the JSON response returned by the MangaDex /at-home/server/{chapterId} endpoint.
type mangaDexAtHomeResponse struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash      string   `json:"hash"`
		Data      []string `json:"data"`
		DataSaver []string `json:"dataSaver"`
	} `json:"chapter"`
}
