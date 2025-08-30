package service

import (
	"encoding/json"
	server "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"time"
)

// Prepare GraphQL mutation
const saveArticle = `
	mutation InsertIntoArticles($objects: [articlesInsertInput!]!) {
		insertIntoarticlesCollection(objects: $objects) {
			records {
				id
				source_url
				category_id
				created_at
				processed_at
				video
				date
			}
		}
	}
`

func SaveArticleToDatabase(content *types.ArticleContent, url string, categoryId int) (*types.Article, error) {
	util.Log("[SaveArticleToDatabase] Saving article to database...", util.LogTypeLog)

	// Prepare variables
	vars := map[string]interface{}{
		"objects": []map[string]interface{}{
			{
				"source_url":   url,
				"category_id":  categoryId,
				"video":        content.Video,
				"date":         content.Date,
				"processed_at": getCurrentTimestampZ(),
			},
		},
	}

	// Call GraphQL
	resp, err := server.CallSupabaseGraphQL(saveArticle, vars)
	if err != nil {
		util.Log("[SaveArticleToDatabase] error calling GraphQL: "+err.Error(), util.LogTypeError)
		return nil, err
	}

	util.Log("[SaveArticleToDatabase] response from saving article: "+string(resp), util.LogTypeLog)

	// Parse response to get the article object
	var result struct {
		Data struct {
			InsertIntoarticlesCollection struct {
				Records []struct {
					ID          string `json:"id"`
					SourceURL   string `json:"source_url"`
					CategoryID  int    `json:"category_id"`
					CreatedAt   string `json:"created_at"`
					ProcessedAt string `json:"processed_at"`
					Video       string `json:"video"`
					Date        string `json:"date"`
				} `json:"records"`
			} `json:"insertIntoarticlesCollection"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		util.Log("[SaveArticleToDatabase] error unmarshalling response: "+err.Error(), util.LogTypeError)
		return nil, err
	}

	if len(result.Data.InsertIntoarticlesCollection.Records) == 0 {
		return nil, nil
	}
	rec := result.Data.InsertIntoarticlesCollection.Records[0]
	article := &types.Article{
		ID:          rec.ID,
		SourceURL:   rec.SourceURL,
		CategoryID:  rec.CategoryID,
		CreatedAt:   rec.CreatedAt,
		ProcessedAt: rec.ProcessedAt,
		Video:       rec.Video,
		Date:        rec.Date,
	}
	return article, nil
}

// getCurrentTimestampZ returns the current timestamp in PostgreSQL timestamptz format
func getCurrentTimestampZ() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05.999999-07")
}
