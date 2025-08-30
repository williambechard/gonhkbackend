package worker

import (
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"strings"
	"sync"
	"time"
)

// ArticleSentenceProcessWorker splits block body into sentences and launches translation workers
func ArticleSentenceProcessWorker(blockTitle, blockBody string) ([]types.ArticleSentenceResult, error) {
	util.Log("[ArticleSentenceProcessWorker] Processing block: Title='"+blockTitle+"'", util.LogTypeLog)
	sentences := strings.Split(blockBody, "。")
	results := make([]types.ArticleSentenceResult, 0, len(sentences))
	errs := make([]error, 0)
	var wg sync.WaitGroup
	resultCh := make(chan types.ArticleSentenceResult, len(sentences))
	errCh := make(chan error, len(sentences))
	sem := make(chan struct{}, 2) // limit to 2 concurrent workers
	for _, sentenceText := range sentences {
		sentenceText = strings.TrimSpace(sentenceText)
		if sentenceText == "" {
			continue
		}
		wg.Add(1)
		go func(s string) {
			sem <- struct{}{}                   // acquire semaphore
			defer func() { <-sem; wg.Done() }() // release semaphore and mark done
			time.Sleep(1 * time.Second)         // delay before each request
			sentenceObj := types.Sentence{JP: s}
			res, err := ArticleSentenceTranslateWorker(sentenceObj)
			if err != nil {
				errCh <- err
			} else {
				resultCh <- res
			}
		}(sentenceText)
	}
	wg.Wait()
	close(resultCh)
	close(errCh)
	for r := range resultCh {
		results = append(results, r)
	}
	for e := range errCh {
		errs = append(errs, e)
	}
	util.Log("[ArticleSentenceProcessWorker] All sentences processed for block: '"+blockTitle+"'", util.LogTypeLog)
	if len(errs) > 0 {
		return results, errs[0] // return first error for simplicity
	}
	return results, nil
}
