package worker

import (
	"nhknewseasybkend/internal/util"
	"strings"
	"sync"
)

// ArticleSentenceProcessWorker splits block body into sentences and launches translation workers
func ArticleSentenceProcessWorker(blockTitle, blockBody string) ([]ArticleSentenceResult, error) {
	util.Log("[ArticleSentenceProcessWorker] Processing block: Title='"+blockTitle+"'", util.LogTypeLog)
	sentences := strings.Split(blockBody, "。")
	results := make([]ArticleSentenceResult, 0, len(sentences))
	errs := make([]error, 0)
	var wg sync.WaitGroup
	resultCh := make(chan ArticleSentenceResult, len(sentences))
	errCh := make(chan error, len(sentences))
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			res, err := ArticleSentenceTranslateWorker(s)
			if err != nil {
				errCh <- err
			} else {
				resultCh <- res
			}
		}(sentence)
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
