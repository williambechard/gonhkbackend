package service

import (
	server "nhknewseasybkend/internal/graphql"
	"nhknewseasybkend/internal/model"
)

func SaveWordsToDB(words []model.AITokenizedWord) error {
	// Batch size for efficient DB writes
	const batchSize = 100

	// Channel to collect errors from goroutines
	errCh := make(chan error, (len(words)/batchSize)+1)

	for i := 0; i < len(words); i += batchSize {
		end := i + batchSize
		if end > len(words) {
			end = len(words)
		}
		batch := words[i:end]

		go func(batch []model.AITokenizedWord) {
			// Define your GraphQL mutation here (to be implemented)
			mutation := `
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
			vars := map[string]interface{}{
				"objects": batch,
			}
			_, err := server.CallSupabaseGraphQL(mutation, vars)
			errCh <- err
		}(batch)
	}

	// Collect errors
	var finalErr error
	for i := 0; i < (len(words)/batchSize)+1; i++ {
		if err := <-errCh; err != nil {
			finalErr = err
		}
	}
	close(errCh)

	return finalErr
}
