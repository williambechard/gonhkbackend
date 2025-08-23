package service

import (
	"encoding/json"
	"fmt"
	server "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/util"
)

type Grammar struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Example     string `json:"example,omitempty"`
	Description string `json:"description,omitempty"`
}

const getGrammarQuery = `
query {
    grammarCollection{
		edges{
			node{
				id
				name
				example
				description
			}
		}
  	}
}`

// Cached list of all grammar points for quick access
var AllGrammar []Grammar

func GetAllGrammarGraphQL() ([]Grammar, error) {
	util.Log("[GrammarService] Fetching all grammar via GraphQL", util.LogTypeLog)
	respBytes, err := server.CallSupabaseGraphQL(getGrammarQuery, nil)
	if err != nil {
		util.Log(fmt.Sprintf("[GrammarService] ERROR calling Supabase GraphQL: %v", err), util.LogTypeError)
		return nil, err
	}

	util.Log(fmt.Sprintf("[GrammarService] RAW GraphQL response: %s", string(respBytes)), util.LogTypeLog)

	var result struct {
		Data struct {
			GrammarCollection struct {
				Edges []struct {
					Node Grammar `json:"node"`
				} `json:"edges"`
			} `json:"grammarCollection"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		util.Log(fmt.Sprintf("[GrammarService] ERROR unmarshaling GraphQL response: %v", err), util.LogTypeError)
		return nil, err
	}

	if len(result.Errors) > 0 {
		errBytes, _ := json.Marshal(result.Errors)
		util.Log(fmt.Sprintf("[GrammarService] GraphQL response contained errors: %s", string(errBytes)), util.LogTypeError)
		return nil, fmt.Errorf("GraphQL errors: %s", string(errBytes))
	}

	var grammarList []Grammar
	for _, edge := range result.Data.GrammarCollection.Edges {
		grammarList = append(grammarList, edge.Node)
	}
	AllGrammar = grammarList
	util.Log(fmt.Sprintf("[GrammarService] Fetched %d grammar points", len(grammarList)), util.LogTypeLog)
	return grammarList, nil
}
