// LinkTranslationWorker runs periodically to process untranslated links from the database.
package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"nhknewseasybkend/internal/util"
	"os"
	"time"
)

const getUntranslatedLinksQuery = `
    query GetUntranslatedLinks {
        linksCollection(filter: { translated: { eq: false } }) {
            edges {
                node {
                    id
                    link
                    translated
					category_id
                }
            }
        }
    }
`

type LinkTranslationWorker struct {
	interval time.Duration
	stopChan chan struct{}
	RunOnce  bool // If true, only run processLinks once (for testing)
}

func NewLinkTranslationWorker(interval time.Duration) *LinkTranslationWorker {
	return &LinkTranslationWorker{
		interval: interval,
		stopChan: make(chan struct{}),
		RunOnce:  true, // default to false, set true for testing
	}
}

func (w *LinkTranslationWorker) Start() {
	go func() {
		if w.RunOnce {
			w.processLinks()
			return
		}
		for {
			select {
			case <-w.stopChan:
				fmt.Println("LinkTranslationWorker stopped.")
				return
			default:
				w.processLinks()
				time.Sleep(w.interval)
			}
		}
	}()
}

func (w *LinkTranslationWorker) Stop() {
	close(w.stopChan)
}

// processLinks would query the DB for untranslated links and process them.
func (w *LinkTranslationWorker) processLinks() {
	payload := map[string]interface{}{
		"query": getUntranslatedLinksQuery,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", os.Getenv("SUPABASE_URL")+"/graphql/v1", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			LinksCollection struct {
				Edges []struct {
					Node struct {
						ID         int    `json:"id"`
						Link       string `json:"link"`
						Translated bool   `json:"translated"`
						CategoryID int    `json:"category_id"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"linksCollection"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		util.Log(fmt.Sprintf("[LinkTranslationWorker] Error decoding response: %v", err), util.LogTypeError)
		return
	}
	count := len(result.Data.LinksCollection.Edges)
	util.Log(fmt.Sprintf("[LinkTranslationWorker] Found %d untranslated links.", count), util.LogTypeLog)

	for i, edge := range result.Data.LinksCollection.Edges {
		if i > 0 {
			break // Stop after processing the first link (for testing)
		}
		link := edge.Node.Link
		util.Log(fmt.Sprintf("[LinkTranslationWorker] Processing link: %s", link), util.LogTypeLog)
		categoryId := edge.Node.CategoryID
		util.Log(fmt.Sprintf("[LinkTranslationWorker] Category ID: %d", categoryId), util.LogTypeLog)
		article, err := ArticleContentWorker(link, categoryId)
		if err != nil {
			util.Log(fmt.Sprintf("[LinkTranslationWorker] Error extracting article for %s: %v", link, err), util.LogTypeError)
			continue
		}
		util.Log(fmt.Sprintf("[LinkTranslationWorker] Extracted article title: %s", article.Title), util.LogTypeLog)
		// Add final log to indicate completion
		if article != nil && len(article.Title) > 0 {
			util.Log(fmt.Sprintf("[LinkTranslationWorker] Finished processing and saving article"), util.LogTypeLog)
		} else {
			util.Log("[LinkTranslationWorker] Finished processing and saving article (ID not available)", util.LogTypeLog)
		}
	}
}
