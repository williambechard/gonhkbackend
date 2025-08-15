package links

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"nhknewseasybkend/internal/config"
	"nhknewseasybkend/internal/util"
)

// Link represents an article link record from Supabase.
type Link struct {
	ID         int    `json:"id"`
	URL        string `json:"url"`
	CategoryID int    `json:"category_id"`
	CreatedAt  string `json:"created_at"`
	// Add other fields as needed based on your Supabase schema
}

var (
	client = &http.Client{}
)

// GetArticleLinksFromSupabase fetches article links from Supabase REST API
func GetLinks(limit, offset int, categoryID *int) ([]*Link, error) {
	maskedUrl := "(not set)"
	if config.SupabaseUrl != "" {
		maskedUrl = "XXXXXXX"
	}
	maskedKey := "(not set)"
	if config.SupabaseKey != "" {
		maskedKey = "XXXXXXX"
	}
	fmt.Printf("SUPABASE_URL: %s, SUPABASE_SERVICE_ROLE_KEY: %s\n", maskedUrl, maskedKey)
	url := fmt.Sprintf("%s/rest/v1/links?select=*&order=created_at.desc&limit=%d&offset=%d", config.SupabaseUrl, limit, offset)
	if categoryID != nil {
		url += fmt.Sprintf("&category_id=eq.%d", *categoryID)
	}
	fmt.Printf("Requesting Supabase URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", config.SupabaseKey)
	req.Header.Set("Authorization", "Bearer "+config.SupabaseKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var links []*Link
	err = json.Unmarshal(body, &links)
	if err != nil {
		return nil, err
	}
	return links, nil
}

// GetArticleLinksByCategoryId fetches article links for a specific category.
func GetLinksByCategoryId(categoryID, limit, offset int) ([]*Link, error) {
	util.Log(fmt.Sprintf("[GetLinksByCategoryId] Called with categoryID=%d, limit=%d, offset=%d", categoryID, limit, offset), util.LogTypeLog)
	return GetLinks(limit, offset, &categoryID)
}
