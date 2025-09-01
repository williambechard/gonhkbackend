package worker

import (
	"fmt"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ArticleSentenceProcessWorker splits block body into sentences and launches translation workers
func ArticleSentenceProcessWorker(blockTitle, blockBody string) ([]types.ArticleSentenceResult, error) {
	util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Processing block: Title length=%d", len(blockTitle)), util.LogTypeLog)
	sentences := strings.Split(blockBody, "。")
	util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Number of sentences to process: %d", len(sentences)), util.LogTypeLog)
	results := make([]types.ArticleSentenceResult, 0, len(sentences))
	errs := make([]error, 0)

	// Configurable batch size via environment variable
	batchSize := 20
	if envBatch := os.Getenv("BATCH_SIZE"); envBatch != "" {
		if parsed, err := strconv.Atoi(envBatch); err == nil && parsed > 0 {
			batchSize = parsed
		}
	}
	const maxTokensPerBatch = 128000 // GPT-4o mini context window
	estimateTokens := func(s string) int {
		// Simple estimation: 1 token per 4 chars (Japanese is compact)
		return (len([]rune(s)) + 3) / 4
	}

	// Split sentences into batches
	var batches [][]string
	var currentBatch []string
	currentTokens := 0
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		tokens := estimateTokens(s)
		if len(currentBatch) >= batchSize || currentTokens+tokens > maxTokensPerBatch {
			batches = append(batches, currentBatch)
			currentBatch = []string{}
			currentTokens = 0
		}
		currentBatch = append(currentBatch, s)
		currentTokens += tokens
	}
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Processing %d batches (batch size: %d)", len(batches), batchSize), util.LogTypeLog)

	// Configurable batch concurrency via environment variable
	batchConcurrency := 2
	if envConc := os.Getenv("BATCH_CONCURRENCY"); envConc != "" {
		if parsed, err := strconv.Atoi(envConc); err == nil && parsed > 0 {
			batchConcurrency = parsed
		}
	}
	batchSem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup
	batchResults := make([][]types.ArticleSentenceResult, len(batches))
	batchErrors := make([]error, len(batches))
	for batchIdx, batch := range batches {
		batchSem <- struct{}{}
		wg.Add(1)
		go func(batchIdx int, batch []string) {
			defer func() { <-batchSem; wg.Done() }()
			util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Processing batch %d of %d (sentences: %d)", batchIdx+1, len(batches), len(batch)), util.LogTypeLog)
			// Batch tokenization
			tokenResults, err := BatchTokenizeSentencesWithAI(batch)
			if err != nil {
				util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Error in batch %d during tokenization: %v", batchIdx+1, err), util.LogTypeError)
				batchErrors[batchIdx] = err
				return
			}
			// Batch enhancement
			enhancedResults, err := BatchEnhanceTokenizedWords(tokenResults)
			if err != nil {
				util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Error in batch %d during enhancement: %v", batchIdx+1, err), util.LogTypeError)
				batchErrors[batchIdx] = err
				return
			}
			// Prepare sentences for translation
			var sentencesForTranslation []types.Sentence
			for i, s := range batch {
				sentencesForTranslation = append(sentencesForTranslation, types.Sentence{JP: s, Words: enhancedResults[i].Words})
			}
			// Batch translation
			translatedResults, err := BatchTranslateJapaneseToEnglish(sentencesForTranslation)
			if err != nil {
				util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Error in batch %d during translation: %v", batchIdx+1, err), util.LogTypeError)
				batchErrors[batchIdx] = err
				return
			}
			// Collect final results
			var batchRes []types.ArticleSentenceResult
			for i := range batch {
				batchRes = append(batchRes, types.ArticleSentenceResult{
					Sentence: translatedResults[i],
					Words:    enhancedResults[i].Words,
				})
			}
			batchResults[batchIdx] = batchRes
			util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] Completed batch %d of %d (sentences: %d)", batchIdx+1, len(batches), len(batch)), util.LogTypeLog)
		}(batchIdx, batch)
	}
	wg.Wait()
	// Collect results and errors in order
	for _, batchRes := range batchResults {
		results = append(results, batchRes...)
	}
	for _, err := range batchErrors {
		if err != nil {
			errs = append(errs, err)
		}
	}
	util.Log(fmt.Sprintf("[ArticleSentenceProcessWorker] All sentences processed for block: Title length=%d", len(blockTitle)), util.LogTypeLog)
	if len(errs) > 0 {
		return results, errs[0] // return first error for simplicity
	}
	return results, nil
}
