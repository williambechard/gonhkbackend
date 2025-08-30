package service

import (
	"encoding/json"
	"fmt"
	server "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"strconv"
)

// LocalWordStore is a simple in-memory dictionary for saved words
var LocalWordStore = make(map[string]types.AITokenizedWord)

var AllWordsQuery = `
query{
	wordsCollection {
		edges {
			node {
				id
				lemma
				lemma_furigana
				confidence
				ai
				jlpt_level
				frequency_rank
				part_of_speech_id
				meanings
			}
		}
	}
}`

var saveSentenceWordsMutation = `
    mutation InsertIntosentence_wordsCollection($objects: [sentence_wordsInsertInput!]!) {
        insertIntosentence_wordsCollection(objects: $objects) {
            records {
                sentence_id
                word_id
                position
                jp
                conjugation
                furigana
                confidence
                contextual_meaning
                grammar_id
            }
        }
    }
`
var saveWordsMutation = `
	mutation InsertWords($objects: [wordsInsertInput!]!) {
		insertIntowordsCollection(objects: $objects) {
			records {
			id
			confidence
			ai
			lemma
			lemma_furigana
			jlpt_level
			frequency_rank
			part_of_speech_id
			meanings
			}
		}
	}
`

func InitWords() {
	util.Log("InitWords: loading all words from DB into LocalWordStore (with pagination)", util.LogTypeLog)
	var allEdges []struct {
		Node types.DBWord `json:"node"`
	}
	var endCursor string
	hasNextPage := true
	pageSize := 100
	for hasNextPage {
		query := fmt.Sprintf(`query{ wordsCollection(first: %d%s) { edges { node { id lemma lemma_furigana confidence ai jlpt_level frequency_rank part_of_speech_id meanings } } pageInfo { hasNextPage endCursor } } }`, pageSize, func() string {
			if endCursor != "" {
				return ", after: \"" + endCursor + "\""
			} else {
				return ""
			}
		}())
		resp, err := server.CallSupabaseGraphQL(query, nil)
		if err != nil {
			util.Log("InitWords: error calling GraphQL: "+err.Error(), util.LogTypeError)
			break
		}
		var result struct {
			Data struct {
				WordsCollection struct {
					Edges []struct {
						Node types.DBWord `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"wordsCollection"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			util.Log("InitWords: error unmarshaling response: "+err.Error(), util.LogTypeError)
			break
		}
		allEdges = append(allEdges, result.Data.WordsCollection.Edges...)
		hasNextPage = result.Data.WordsCollection.PageInfo.HasNextPage
		endCursor = result.Data.WordsCollection.PageInfo.EndCursor
	}
	for _, edge := range allEdges {
		token := edge.Node.Lemma
		if token == "" {
			util.Log("InitWords: DBWord missing Lemma, skipping", util.LogTypeWarn)
			continue
		}
		LocalWordStore[token] = types.AITokenizedWord{
			Token:          token,
			Furigana:       edge.Node.LemmaFurigana,
			Lemma:          edge.Node.Lemma,
			LemmaFurigana:  edge.Node.LemmaFurigana,
			PartOfSpeechID: edge.Node.PartOfSpeechID,
			JLPTLevel:      edge.Node.JLPTLevel,
			FrequencyRank:  edge.Node.FrequencyRank,
			EnglishMeaning: edge.Node.Meanings,
		}
	}
	util.Log(fmt.Sprintf("InitWords: loaded words: %d", len(allEdges)), util.LogTypeLog)
	return
}

// AddWordsToLocalStore adds an array of AITokenizedWord to LocalWordStore
func AddWordsToLocalStore(words []types.AITokenizedWord) {
	for _, w := range words {
		LocalWordStore[w.Token] = w
	}
}

func SaveSentenceWordsToDB(sentence types.Sentence) error {
	if len(sentence.Words) == 0 || sentence.Words[0].ID == "" {
		util.Log("SaveSentenceWordsToDB: First word in Words[] does not have a valid ID. Exiting.", util.LogTypeError)
		panic("SaveSentenceWordsToDB: First word in Words[] does not have a valid ID. Exiting.")
	}
	util.Log("SaveSentenceWordsToDB: saving sentence_words for sentence: "+sentence.EN, util.LogTypeLog)

	var objects []map[string]interface{}
	toInt := func(val interface{}) int {
		switch v := val.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case string:
			var i int
			fmt.Sscanf(v, "%d", &i)
			return i
		default:
			return 0
		}
	}
	for i, w := range sentence.Words {
		wordID := toInt(w.ID)
		sentenceID := toInt(sentence.ID)
		existsInStore := false
		if local, exists := LocalWordStore[w.Token]; exists {
			existsInStore = true
			util.Log(fmt.Sprintf("[SaveSentenceWordsToDB] LocalWordStore: token='%s', id='%v'", w.Token, local.ID), util.LogTypeLog)
		} else {
			util.Log(fmt.Sprintf("[SaveSentenceWordsToDB] LocalWordStore: token='%s' NOT FOUND", w.Token), util.LogTypeWarn)
		}
		util.Log(fmt.Sprintf("[SaveSentenceWordsToDB] wordID=%v, token='%s', existsInStore=%v", wordID, w.Token, existsInStore), util.LogTypeLog)
		if wordID == 0 {
			util.Log(fmt.Sprintf("[SaveSentenceWordsToDB] WARNING: wordID is zero for token '%s' (sentenceID=%d, position=%d)", w.Token, sentenceID, i), util.LogTypeWarn)
		}
		objects = append(objects, map[string]interface{}{
			"sentence_id":        sentenceID,
			"word_id":            wordID,
			"position":           i,
			"jp":                 w.Token,
			"conjugation":        w.Conjugation,
			"furigana":           w.Furigana,
			"confidence":         fmt.Sprintf("%.2f", w.Confidence),
			"contextual_meaning": w.EnglishMeaning,
			"grammar_id":         1, //w.GrammarID
		})
		util.Log(fmt.Sprintf("[SaveSentenceWordsToDB] Prepared object: sentence_id=%d, word_id=%d, token='%s', position=%d", sentenceID, wordID, w.Token, i), util.LogTypeLog)
	}

	var vars = map[string]interface{}{
		"objects": objects,
	}
	util.Log("SaveSentenceWordsToDB: mutation variables: "+fmt.Sprintf("%#v", vars), util.LogTypeLog)
	resp, err := server.CallSupabaseGraphQL(saveSentenceWordsMutation, vars)
	util.Log("SaveSentenceWordsToDB: raw GraphQL response: "+string(resp), util.LogTypeLog)
	if err != nil {
		util.Log("SaveSentenceWordsToDB: error calling GraphQL: "+err.Error(), util.LogTypeError)
		return err
	}
	util.Log("SaveSentenceWordsToDB: saved sentence_words for sentence_id "+sentence.ID, util.LogTypeLog)
	return nil
}

func SaveWordsToDB(words []types.AITokenizedWord) ([]types.AITokenizedWord, error) {
	// Batch size for efficient DB writes
	const batchSize = 100

	// Channel to collect errors from goroutines
	errCh := make(chan error, (len(words)/batchSize)+1)
	// Channel to collect updated words from goroutines
	updatedWordsCh := make(chan []types.AITokenizedWord, (len(words)/batchSize)+1)

	// Check LocalWordStore for existing words and set their IDs
	var wordsToSave []types.AITokenizedWord
	for i, w := range words {
		if w.ID == "" {
			if local, exists := LocalWordStore[w.Token]; exists && local.ID != "" {
				// Word exists in DB, set its ID
				words[i].ID = local.ID
				util.Log(fmt.Sprintf("[SaveWordsToDB] Word '%s' already exists in DB with ID %v", w.Token, local.ID), util.LogTypeLog)
			} else {
				// Word not in DB, needs to be saved
				wordsToSave = append(wordsToSave, w)
				util.Log(fmt.Sprintf("[SaveWordsToDB] Will save new word: token='%s', lemma='%s'", w.Token, w.Lemma), util.LogTypeLog)
			}
		}
	}
	util.Log(fmt.Sprintf("[SaveWordsToDB] Total new words to save: %d", len(wordsToSave)), util.LogTypeLog)
	if len(wordsToSave) == 0 {
		util.Log("[SaveWordsToDB] No new words to save. Returning immediately.", util.LogTypeLog)
		// Update the original words slice with IDs from LocalWordStore
		for i, w := range words {
			if local, exists := LocalWordStore[w.Token]; exists && local.ID != "" {
				words[i].ID = local.ID
			}
		}
		return words, nil
	}

	for i := 0; i < len(wordsToSave); i += batchSize {
		end := i + batchSize
		if end > len(wordsToSave) {
			end = len(wordsToSave)
		}
		batch := wordsToSave[i:end]

		util.Log(fmt.Sprintf("[SaveWordsToDB] Sending batch of %d words to DB", len(batch)), util.LogTypeLog)
		for _, w := range batch {
			util.Log(fmt.Sprintf("[SaveWordsToDB] Batch word: token='%s', lemma='%s'", w.Token, w.Lemma), util.LogTypeLog)
		}

		go func(batch []types.AITokenizedWord) {
			var objects []map[string]interface{}
			for _, w := range batch {
				jlptLevel := 0
				if w.JLPTLevel != nil {
					jlptLevel = *w.JLPTLevel
				}
				freqRank := 0
				if w.FrequencyRank != nil {
					freqRank = *w.FrequencyRank
				}
				objects = append(objects, map[string]interface{}{
					"lemma":             w.Lemma,
					"lemma_furigana":    w.LemmaFurigana,
					"confidence":        fmt.Sprintf("%.2f", w.Confidence),
					"ai":                true,
					"jlpt_level":        jlptLevel,
					"frequency_rank":    freqRank,
					"part_of_speech_id": w.PartOfSpeechID,
					"meanings":          w.EnglishMeaning,
				})
			}
			varrs := map[string]interface{}{
				"objects": objects,
			}
			util.Log("SaveWordsToDB: mutation variables: "+fmt.Sprintf("%#v", varrs), util.LogTypeLog)
			resp, err := server.CallSupabaseGraphQL(saveWordsMutation, varrs)
			util.Log("SaveWordsToDB: raw GraphQL response: "+string(resp), util.LogTypeLog)
			var updatedBatch []types.AITokenizedWord
			if err == nil {
				var result struct {
					Data struct {
						InsertIntowordsCollection struct {
							Records []struct {
								ID             string   `json:"id"`
								Lemma          string   `json:"lemma"`
								LemmaFurigana  string   `json:"lemma_furigana"`
								Confidence     string   `json:"confidence"`
								AI             bool     `json:"ai"`
								JLPTLevel      int      `json:"jlpt_level"`
								FrequencyRank  int      `json:"frequency_rank"`
								PartOfSpeechID int      `json:"part_of_speech_id"`
								Meanings       []string `json:"meanings"`
							} `json:"records"`
						} `json:"insertIntowordsCollection"`
					} `json:"data"`
				}
				if err := json.Unmarshal(resp, &result); err == nil {
					for _, word := range result.Data.InsertIntowordsCollection.Records {
						conf, _ := strconv.ParseFloat(word.Confidence, 64)
						w := types.AITokenizedWord{
							ID:             word.ID,
							Lemma:          word.Lemma,
							LemmaFurigana:  word.LemmaFurigana,
							Confidence:     conf,
							JLPTLevel:      &word.JLPTLevel,
							FrequencyRank:  &word.FrequencyRank,
							PartOfSpeechID: word.PartOfSpeechID,
							EnglishMeaning: word.Meanings,
						}
						util.Log(fmt.Sprintf("[SaveWordsToDB] DB returned word: token='%s', id='%v'", w.Lemma, w.ID), util.LogTypeLog)
						if _, exists := LocalWordStore[w.Lemma]; !exists {
							LocalWordStore[w.Lemma] = w
						}
						updatedBatch = append(updatedBatch, w)
					}
				} else {
					util.Log(fmt.Sprintf("[SaveWordsToDB] ERROR unmarshaling DB response: %v", err), util.LogTypeError)
					panic(fmt.Sprintf("SaveWordsToDB: ERROR unmarshaling DB response: %v", err))
				}
			} else {
				util.Log(fmt.Sprintf("[SaveWordsToDB] ERROR from DB: %v", err), util.LogTypeError)
				panic(fmt.Sprintf("SaveWordsToDB: ERROR from DB: %v", err))
			}
			updatedWordsCh <- updatedBatch
			errCh <- err
		}(batch)
	}

	util.Log("[SaveWordsToDB] All batches dispatched", util.LogTypeLog)

	// Collect errors
	var finalErr error
	var updatedWords []types.AITokenizedWord
	for i := 0; i < (len(words)/batchSize)+1; i++ {
		if err := <-errCh; err != nil {
			finalErr = err
		}
		batchWords := <-updatedWordsCh
		for _, uw := range batchWords {
			updatedWords = append(updatedWords, uw)
		}
	}
	close(errCh)
	close(updatedWordsCh)

	// Update the original words slice with IDs
	wordMap := make(map[string]types.AITokenizedWord)
	for _, uw := range updatedWords {
		wordMap[uw.Token] = uw
	}
	for i, w := range words {
		if uw, ok := wordMap[w.Token]; ok {
			words[i].ID = uw.ID
		}
	}

	util.Log(fmt.Sprintf("[SaveWordsToDB] Total words saved/updated: %d", len(updatedWords)), util.LogTypeLog)

	AddWordsToLocalStore(updatedWords)
	return updatedWords, finalErr
}
