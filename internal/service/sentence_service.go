package service

import (
	"encoding/json"
	"fmt"
	server "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
)

const saveSentenceMutation = `
	mutation InsertSentence($objects: [sentencesInsertInput!]!) {
		insertIntosentencesCollection(objects: $objects) {
			records {
				id
				jp
				en
				eval
			}
		}
	}
`

const saveArticleSentenceMutation = `
	mutation InsertArticleSentence($objects: [article_sentencesInsertInput!]!) {
		insertIntoarticle_sentencesCollection(objects: $objects) {
			records {
				id
				article_id
				sentence_id
				part
				sub_index
				position
				media_url
			}
		}
	}
`

const getSentenceQuery = `
query GetSentence($jp: String!, $en: String!, $eval: String!) {
  sentencesCollection(filter: { jp: { eq: $jp }, en: { eq: $en }, eval: { eq: $eval } }) {
    edges {
      node {
        id
        jp
        en
        eval
      }
    }
  }
}`

// GetSentenceFromDB checks for an existing sentence in the DB
func GetSentenceFromDB(jp, en, eval string) (*types.Sentence, error) {
	vars := map[string]interface{}{
		"jp":   jp,
		"en":   en,
		"eval": eval,
	}
	resp, err := server.CallSupabaseGraphQL(getSentenceQuery, vars)
	if err != nil {
		util.Log("GetSentenceFromDB: error calling GraphQL: "+err.Error(), util.LogTypeError)
		return nil, err
	}
	var result struct {
		Data struct {
			SentencesCollection struct {
				Edges []struct {
					Node struct {
						ID   string `json:"id"`
						JP   string `json:"jp"`
						EN   string `json:"en"`
						Eval string `json:"eval"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"sentencesCollection"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		util.Log("GetSentenceFromDB: error unmarshaling response: "+err.Error(), util.LogTypeError)
		return nil, err
	}
	if len(result.Data.SentencesCollection.Edges) == 0 {
		return nil, nil
	}
	node := result.Data.SentencesCollection.Edges[0].Node
	return &types.Sentence{ID: node.ID, JP: node.JP, EN: node.EN, Eval: node.Eval}, nil
}

// SaveSentenceToDB saves a sentence to the DB using GraphQL and returns the Sentence object with the new id
func SaveSentenceToDB(jp, en, eval string) (*types.Sentence, error) {
	util.Log("SaveSentenceToDB: saving sentence to DB", util.LogTypeLog)

	// Uniqueness check
	existing, err := GetSentenceFromDB(jp, en, eval)
	if err != nil {
		util.Log("SaveSentenceToDB: error checking for existing sentence: "+err.Error(), util.LogTypeError)
		return nil, err
	}
	if existing != nil {
		util.Log("SaveSentenceToDB: found existing sentence, not inserting duplicate", util.LogTypeLog)
		return existing, nil
	}

	vars := map[string]interface{}{
		"objects": []map[string]interface{}{
			{
				"jp":   jp,
				"en":   en,
				"eval": eval,
			},
		},
	}
	resp, err := server.CallSupabaseGraphQL(saveSentenceMutation, vars)
	if err != nil {
		util.Log("SaveSentenceToDB: error calling GraphQL: "+err.Error(), util.LogTypeError)
		return nil, err
	}
	var result struct {
		Data struct {
			InsertIntosentencesCollection struct {
				Records []struct {
					ID   string `json:"id"`
					JP   string `json:"jp"`
					EN   string `json:"en"`
					Eval string `json:"eval"`
				} `json:"records"`
			} `json:"insertIntosentencesCollection"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		util.Log("SaveSentenceToDB: error unmarshaling response: "+err.Error(), util.LogTypeError)
		return nil, err
	}
	if len(result.Data.InsertIntosentencesCollection.Records) == 0 {
		util.Log("SaveSentenceToDB: no records returned", util.LogTypeWarn)
		return nil, fmt.Errorf("no records returned")
	}
	record := result.Data.InsertIntosentencesCollection.Records[0]
	sentence := &types.Sentence{
		ID:   record.ID,
		JP:   record.JP,
		EN:   record.EN,
		Eval: record.Eval,
	}
	util.Log("SaveSentenceToDB: sentence saved successfully", util.LogTypeLog)
	return sentence, nil
}

// SaveArticleSentenceToDB saves an article-sentence association to the DB using GraphQL
func SaveArticleSentenceToDB(articleID string, sentenceID int, part string, subIndex, position int, mediaURL string) error {
	util.Log("SaveArticleSentenceToDB: saving article-sentence association to DB", util.LogTypeLog)
	util.Log(fmt.Sprintf("SaveArticleSentenceToDB: input articleID=%d, sentenceID=%d, part=%s, subIndex=%d, position=%d, mediaURL=%s", articleID, sentenceID, part, subIndex, position, mediaURL), util.LogTypeLog)
	vars := map[string]interface{}{
		"objects": []map[string]interface{}{
			{
				"article_id":  util.StringToInt(articleID),
				"sentence_id": sentenceID,
				"part":        part,
				"sub_index":   subIndex,
				"position":    position,
				"media_url":   mediaURL,
			},
		},
	}
	util.Log("SaveArticleSentenceToDB: mutation variables: "+fmt.Sprintf("%#v", vars), util.LogTypeLog)
	resp, err := server.CallSupabaseGraphQL(saveArticleSentenceMutation, vars)
	if err != nil {
		util.Log("SaveArticleSentenceToDB: error calling GraphQL: "+err.Error(), util.LogTypeError)
		return err
	}
	util.Log("SaveArticleSentenceToDB: raw GraphQL response: "+string(resp), util.LogTypeLog)
	var result struct {
		Data struct {
			InsertIntoarticle_sentencesCollection struct {
				Records []struct {
					ID         int    `json:"id"`
					ArticleID  int    `json:"article_id"`
					SentenceID int    `json:"sentence_id"`
					Part       string `json:"part"`
					SubIndex   int    `json:"sub_index"`
					Position   int    `json:"position"`
					MediaURL   string `json:"media_url"`
				} `json:"records"`
			} `json:"insertIntoarticle_sentencesCollection"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		util.Log("SaveArticleSentenceToDB: error unmarshaling response: "+err.Error(), util.LogTypeError)
		return err
	}
	if len(result.Data.InsertIntoarticle_sentencesCollection.Records) == 0 {
		util.Log("SaveArticleSentenceToDB: no records returned. Variables: "+fmt.Sprintf("%#v", vars), util.LogTypeError)
		util.Log("SaveArticleSentenceToDB: raw response: "+string(resp), util.LogTypeError)
		return fmt.Errorf("no records returned")
	}
	util.Log("SaveArticleSentenceToDB: association saved successfully", util.LogTypeLog)
	return nil
}
