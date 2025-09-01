package service

import (
	"encoding/json"
	"fmt"
	server "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/util"
)

type PartOfSpeech struct {
	ID       string `json:"id"`
	Eng      string `json:"eng"`
	Pos      string `json:"pos"`
	Furigana string `json:"furigana,omitempty"`
	Abr      string `json:"abr,omitempty"`
}

// Cached list of all parts of speech for quick access
var AllPartsOfSpeech []PartOfSpeech

const getPartsOfSpeechQuery = `
  query GetPartsOfSpeech {
	 parts_of_speechCollection {
		 edges {
			 node {
				 id
				 eng
				 pos
				 furigana
				 abr
			 }
		 }
	 }
 }
`

func GetAllPartsOfSpeechGraphQL() ([]PartOfSpeech, error) {
	util.Log("[PartsOfSpeechService] Fetching all parts of speech via GraphQL", util.LogTypeLog)
	respBytes, err := server.CallSupabaseGraphQL(getPartsOfSpeechQuery, nil)
	if err != nil {
		util.Log(fmt.Sprintf("[PartsOfSpeechService] ERROR calling Supabase GraphQL: %v", err), util.LogTypeError)
		return nil, err
	}

	// Log the raw response for debugging
	util.Log(fmt.Sprintf("[PartsOfSpeechService] RAW GraphQL response length: %d", len(respBytes)), util.LogTypeLog)

	var result struct {
		Data struct {
			PartsOfSpeechCollection struct {
				Edges []struct {
					Node PartOfSpeech `json:"node"`
				} `json:"edges"`
			} `json:"parts_of_speechCollection"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		util.Log(fmt.Sprintf("[PartsOfSpeechService] ERROR unmarshaling GraphQL response: %v", err), util.LogTypeError)
		return nil, err
	}

	if len(result.Errors) > 0 {
		errBytes, _ := json.Marshal(result.Errors)
		util.Log(fmt.Sprintf("[PartsOfSpeechService] GraphQL response contained %d errors", len(result.Errors)), util.LogTypeError)
		return nil, fmt.Errorf("GraphQL errors: %s", string(errBytes))
	}

	var posList []PartOfSpeech
	for _, edge := range result.Data.PartsOfSpeechCollection.Edges {
		posList = append(posList, edge.Node)
	}
	// Cache the results for later use
	AllPartsOfSpeech = posList
	util.Log(fmt.Sprintf("[PartsOfSpeechService] Fetched %d parts of speech", len(posList)), util.LogTypeLog)
	return posList, nil
}
