package worker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"nhknewseasybkend/internal/service"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Enhanced error logging for early returns in ArticleContentProcessWorker
func logEarlyReturn(context string, err error) {
	if err != nil {
		util.Log(fmt.Sprintf("[ArticleContentProcessWorker] EARLY RETURN: %s: %v", context, err), util.LogTypeError)
	} else {
		util.Log(fmt.Sprintf("[ArticleContentProcessWorker] EARLY RETURN: %s", context), util.LogTypeError)
	}
}

// ExtractUniqueWordsFromArticleContent returns a slice of unique AITokenizedWord from the entire ArticleContent
func ExtractUniqueWordsFromArticleContent(article *types.ArticleContent) []types.AITokenizedWord {
	unique := make(map[string]types.AITokenizedWord)
	// Helper to process a slice of Sentence
	processSentences := func(sentences []types.Sentence) {
		for _, s := range sentences {
			for _, w := range s.Words {
				if _, exists := unique[w.Token]; !exists {
					unique[w.Token] = w
				}
			}
		}
	}
	processSentences(article.Title)
	processSentences(article.Summary)
	for _, block := range article.Content {
		processSentences(block.Title)
		processSentences(block.Body)
	}
	// Convert map to slice
	result := make([]types.AITokenizedWord, 0, len(unique))
	for _, w := range unique {
		result = append(result, w)
	}
	return result
}

// ArticleContentWorker runs periodically to process article URLs
// and extract Japanese sentences/content
// You can expand this to accept a list of URLs or integrate with your DB
func ArticleContentWorker(url string, categoryId int) (*types.ArticleContent, error) {
	util.Log(fmt.Sprintf("[ArticleContentWorker] Fetching article from URL: %s", url), util.LogTypeLog)

	// Check for cached ArticleContent JSON
	jsonPath := "article_content_cache.json"
	content := &types.ArticleContent{}
	if _, err := os.Stat(jsonPath); err == nil {
		util.Log("[ArticleContentWorker] Found cached article_content_cache.json, loading...", util.LogTypeLog)
		jsonBytes, err := os.ReadFile(jsonPath)
		if err != nil {
			util.Log(fmt.Sprintf("[ArticleContentWorker] Error reading cache file: %v", err), util.LogTypeError)
			return nil, err
		}
		var cachedContent types.ArticleContent
		if err := json.Unmarshal(jsonBytes, &cachedContent); err != nil {
			util.Log(fmt.Sprintf("[ArticleContentWorker] Error unmarshaling cache JSON: %v", err), util.LogTypeError)
			return nil, err
		}
		util.Log("[ArticleContentWorker] Loaded ArticleContent from cache, skipping extraction.", util.LogTypeLog)
		content = &cachedContent
	} else {
		util.Log("[ArticleContentWorker] No cached content found, proceeding with extraction.", util.LogTypeLog)

		// Launch browser with Rod
		l := launcher.New().Headless(true).NoSandbox(true).MustLaunch()
		util.Log(fmt.Sprintf("[ArticleContentWorker] Using browser executable: %s", l), util.LogTypeLog)
		browser := rod.New().ControlURL(l).MustConnect()
		defer browser.MustClose()

		page := browser.MustPage(url)
		util.Log("[ArticleContentWorker] Navigated to URL with Rod.", util.LogTypeLog)
		// Wait for body to be loaded
		err := page.WaitLoad()
		if err != nil {
			util.Log(fmt.Sprintf("[ArticleContentWorker] ERROR during page.WaitLoad: %v", err), util.LogTypeError)
			return nil, err
		}
		util.Log("[ArticleContentWorker] Page loaded.", util.LogTypeLog)
		html, err := page.HTML()
		if err != nil {
			util.Log(fmt.Sprintf("[ArticleContentWorker] ERROR getting page HTML: %v", err), util.LogTypeError)
			return nil, err
		}
		util.Log(fmt.Sprintf("[ArticleContentWorker] Extracted HTML length: %d", len(html)), util.LogTypeLog)

		// Parse HTML with goquery
		util.Log("[ArticleContentWorker] Invoking parseArticleHTML...", util.LogTypeLog)
		content, err := parseArticleHTML(html, url)
		if err != nil {
			util.Log(fmt.Sprintf("[ArticleContentWorker] ERROR during HTML parsing: %v", err), util.LogTypeError)
			return nil, err
		}
		util.Log(fmt.Sprintf("[ArticleContentWorker] Article extraction completed successfully for %s", url), util.LogTypeLog)

		// Save ArticleContent struct as JSON for future reuse
		jsonBytes, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			util.Log(fmt.Sprintf("[ArticleContentWorker] Error marshaling ArticleContent to JSON: %v", err), util.LogTypeError)
		} else {
			jsonPath := "article_content_cache.json" // Save at project root
			err = os.WriteFile(jsonPath, jsonBytes, 0644)
			if err != nil {
				util.Log(fmt.Sprintf("[ArticleContentWorker] Error writing ArticleContent JSON to file: %v", err), util.LogTypeError)
			} else {
				util.Log(fmt.Sprintf("[ArticleContentWorker] ArticleContent JSON saved to %s", jsonPath), util.LogTypeLog)
			}
		}

		os.Exit(0)
	}

	util.Log("[ArticleContentWorker] EXIT: ArticleContent processing completed. Moving to Saving to DB.", util.LogTypeLog)

	// Save Article to database
	article, errSaveArticle := service.SaveArticleToDatabase(content, url, categoryId)
	if errSaveArticle != nil {
		util.Log(fmt.Sprintf("[ArticleContentWorker] Error saving article to database: %v", errSaveArticle), util.LogTypeError)
	}
	// Assume SaveArticleToDatabase returns the new article ID (implement as needed)
	articleID := article.ID // Replace with actual article ID from DB if available

	util.Log(fmt.Sprintf("[ArticleContentWorker] Article saved to database with ID: %d", articleID), util.LogTypeLog)

	position := 0

	util.Log(fmt.Sprintf("[ArticleContentWorker] Starting to save sentences and words for article ID: %d", articleID), util.LogTypeLog)

	// Title sentences
	for subIdx, s := range content.Title {
		util.Log(fmt.Sprintf("[ArticleContentWorker] Looping through Title: %s", s.JP), util.LogTypeLog)
		if err := saveSentenceAndWords(s, "title", position, subIdx, "", toInt(articleID)); err != nil {
			return nil, err
		}
		util.Log(fmt.Sprintf("[ArticleContentWorker] Saved title sentence for article ID: %d, position: %d, subIndex: %d", articleID, position, subIdx), util.LogTypeLog)
		position++
	}
	// Summary sentences
	for subIdx, s := range content.Summary {
		util.Log(fmt.Sprintf("[ArticleContentWorker] Looping through Summary: %s", s.JP), util.LogTypeLog)
		if err := saveSentenceAndWords(s, "summary", position, subIdx, "", toInt(articleID)); err != nil {
			return nil, err
		}
		util.Log(fmt.Sprintf("[ArticleContentWorker] Saved summary sentence for article ID: %d, position: %d, subIndex: %d", articleID, position, subIdx), util.LogTypeLog)
		position++
	}
	// Content blocks
	for _, block := range content.Content {

		// Block title sentences
		for subIdx, s := range block.Title {
			util.Log(fmt.Sprintf("[ArticleContentWorker] Looping through Block Title: %s", s.JP), util.LogTypeLog)
			if err := saveSentenceAndWords(s, "block_title", position, subIdx, block.Img, toInt(articleID)); err != nil {
				return nil, err
			}
			util.Log(fmt.Sprintf("[ArticleContentWorker] Saved block title sentence for article ID: %d, position: %d, subIndex: %d", articleID, position, subIdx), util.LogTypeLog)
			position++
		}
		// Block body sentences
		for subIdx, s := range block.Body {
			util.Log(fmt.Sprintf("[ArticleContentWorker] Looping through Block Body: %s", s.JP), util.LogTypeLog)
			if err := saveSentenceAndWords(s, "block_body", position, subIdx, block.Img, toInt(articleID)); err != nil {
				return nil, err
			}
			util.Log(fmt.Sprintf("[ArticleContentWorker] Saved block body sentence for article ID: %d, position: %d, subIndex: %d", articleID, position, subIdx), util.LogTypeLog)
			position++
		}
	}

	util.Log(fmt.Sprintf("[ArticleContentWorker] All sentences and words saved for article ID: %d", articleID), util.LogTypeLog)

	return content, nil
}

func saveSentenceAndWords(s types.Sentence, part string, position, subIndex int, mediaURL string, articleID int) error {
	util.Log(fmt.Sprintf("[ArticleContentWorker] Saving sentence and words for article ID: %d, part: %s, position: %d, subIndex: %d", articleID, part, position, subIndex), util.LogTypeLog)
	// Map part to allowed values
	allowedPart := part
	switch part {
	case "block_title":
		allowedPart = "title"
	case "block_body":
		allowedPart = "body"
	}

	// Save words and update IDs in sentence.Words
	savedWords, err := service.SaveWordsToDB(s.Words)
	if err != nil {
		util.Log("SaveSentenceAndWords: error saving words: "+err.Error(), util.LogTypeError)
		return err
	}

	util.Log(fmt.Sprintf("[saveSentenceAndWords] savedWords: %v", savedWords), util.LogTypeLog)

	for idx, w := range savedWords {
		util.Log(fmt.Sprintf("[saveSentenceAndWords] savedWords[%d]: token='%s', id='%v'", idx, w.Token, w.ID), util.LogTypeLog)
	}

	// Save sentence to DB
	savedSentence, err := service.SaveSentenceToDB(s.JP, s.EN, s.Eval)
	if err != nil {
		util.Log("SaveSentenceAndWords: error saving sentence: "+err.Error(), util.LogTypeError)
		return err
	}
	if savedSentence == nil {
		util.Log("SaveSentenceAndWords: nil sentence returned", util.LogTypeError)
		return fmt.Errorf("nil sentence returned")
	}
	savedSentence.Words = savedWords

	// Save sentence-word associations
	err = service.SaveSentenceWordsToDB(*savedSentence)
	if err != nil {
		util.Log("SaveSentenceAndWords: error saving sentence words: "+err.Error(), util.LogTypeError)
		return err
	}

	// Save article-sentence association
	err = service.SaveArticleSentenceToDB(strconv.Itoa(articleID), toInt(savedSentence.ID), allowedPart, subIndex, position, mediaURL)
	if err != nil {
		util.Log("SaveSentenceAndWords: error saving article-sentence association: "+err.Error(), util.LogTypeError)
		return err
	}
	return nil
}

// Helper to convert string ID to int
func toInt(id string) int {
	i, _ := strconv.Atoi(id)
	return i
}

// parseArticleHTML parses the HTML and extracts article content
func parseArticleHTML(html, baseURL string) (*types.ArticleContent, error) {
	util.Log("[ArticleContentWorker] Starting HTML parsing...", util.LogTypeLog)
	if html == "" {
		util.Log("[ArticleContentWorker] HTML is empty!", util.LogTypeError)
		return nil, fmt.Errorf("empty HTML")
	}

	util.Log("[ArticleContentWorker] Creating goquery document...", util.LogTypeLog)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		util.Log(fmt.Sprintf("[ArticleContentWorker] Error creating goquery document: %v", err), util.LogTypeError)
		return nil, err
	}
	util.Log("[ArticleContentWorker] goquery document created successfully.", util.LogTypeLog)

	titleRaw := strings.TrimSpace(doc.Find(".content--title span").Text())
	video := doc.Find(".content--video iframe, .content--video video").AttrOr("src", "")
	date := doc.Find(".content--date time").AttrOr("datetime", "")
	summaryRaw := strings.TrimSpace(doc.Find(".content--summary").Text())

	util.Log(fmt.Sprintf("[ArticleContentWorker] Extracted title: '%s'", titleRaw), util.LogTypeLog)
	util.Log(fmt.Sprintf("[ArticleContentWorker] Extracted video: '%s'", video), util.LogTypeLog)
	util.Log(fmt.Sprintf("[ArticleContentWorker] Extracted date: '%s'", date), util.LogTypeLog)
	util.Log(fmt.Sprintf("[ArticleContentWorker] Extracted summary: '%s'", summaryRaw), util.LogTypeLog)

	// Tokenize title and summary
	titleSentences, _ := processContent("title", titleRaw)
	summarySentences, _ := processContent("summary", summaryRaw)

	var contentBlocks []types.ContentBlock

	blockCount := 0
	util.Log("[ArticleContentWorker] Searching for content blocks...", util.LogTypeLog)
	doc.Find(".content--detail-more section.content--body").Each(func(i int, section *goquery.Selection) {
		blockTitleRaw := strings.TrimSpace(section.Find(".body-title").Text())
		imgRaw := section.Find("figure.body-image img").AttrOr("src", "")
		if imgRaw == "" {
			imgRaw = section.Find("figure.body-image img").AttrOr("data-src", "")
		}
		if imgRaw == "" {
			imgRaw = section.Find("figure.body-img img").AttrOr("src", "")
		}
		if imgRaw == "" {
			imgRaw = section.Find("figure.body-img img").AttrOr("data-src", "")
		}
		if imgRaw == "" {
			imgRaw = section.Find("figure.body-img.is-fluid img").AttrOr("src", "")
		}
		if imgRaw == "" {
			imgRaw = section.Find("figure.body-img.is-fluid img").AttrOr("data-src", "")
		}
		// Remove default image
		img := ""
		if imgRaw != "" && !strings.Contains(imgRaw, "noimg_default.gif") {
			if strings.HasPrefix(imgRaw, "/") {
				u, err := url.Parse(baseURL)
				if err == nil {
					img = u.Scheme + "://" + u.Host + imgRaw
				} else {
					img = imgRaw
				}
			} else {
				img = imgRaw
			}
		}
		var paragraphs []string
		section.Find(".body-text p").Each(func(j int, p *goquery.Selection) {
			txt := strings.TrimSpace(p.Text())
			if txt != "" {
				paragraphs = append(paragraphs, txt)
			}
		})
		bodyRaw := strings.Join(paragraphs, "\n")
		// Tokenize block title and body
		blockTitleSentences, _ := processContent("blockTitle", blockTitleRaw)
		bodySentences, _ := processContent("body", bodyRaw)
		if bodyRaw != "" {
			contentBlocks = append(contentBlocks, types.ContentBlock{
				Title: blockTitleSentences,
				Img:   img,
				Body:  bodySentences,
			})
			blockCount++
			util.Log(fmt.Sprintf("[ArticleContentWorker] Extracted block #%d: title='%s', img='%s', body length=%d", blockCount, blockTitleRaw, img, len(bodyRaw)), util.LogTypeLog)
		}
	})

	util.Log(fmt.Sprintf("[ArticleContentWorker] Total content blocks extracted: %d", blockCount), util.LogTypeLog)

	return &types.ArticleContent{
		Title:   titleSentences,
		Video:   video,
		Date:    date,
		Summary: summarySentences,
		Content: contentBlocks,
	}, nil
}

// Updated processContent to call ArticleSentenceProcessWorker directly
func processContent(title, body string) ([]types.Sentence, error) {
	util.Log(fmt.Sprintf("[processContent] Processing block: Title='%s', Body length=%d", title, len(body)), util.LogTypeLog)
	results, err := ArticleSentenceProcessWorker(title, body)
	util.Log(fmt.Sprintf("[processContent] Finished processing block: Title='%s'", title), util.LogTypeLog)
	sentences := make([]types.Sentence, 0, len(results))
	for _, r := range results {
		sentences = append(sentences, types.Sentence{JP: r.Sentence.JP, EN: r.Sentence.EN, Eval: r.Sentence.Eval, Words: r.Sentence.Words})
	}
	return sentences, err
}

// ArticleContentProcessWorker processes the full ArticleContent struct
func ArticleContentProcessWorker(article *types.ArticleContent, articleID int, done chan<- bool) {
	util.Log("[ArticleContentProcessWorker] TEST ENTRY -- function called", util.LogTypeLog)
	if article == nil {
		util.Log("[ArticleContentProcessWorker] EARLY RETURN: article is nil", util.LogTypeError)
		done <- true
		return
	}
	titleJPs := make([]string, len(article.Title))
	for i, s := range article.Title {
		titleJPs[i] = s.JP
	}
	util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Title: '%s'", strings.Join(titleJPs, "; ")), util.LogTypeLog)
	summaryJPs := make([]string, len(article.Summary))
	for i, s := range article.Summary {
		summaryJPs[i] = s.JP
	}
	util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Summary: '%s'", strings.Join(summaryJPs, "; ")), util.LogTypeLog)

	for i, block := range article.Content {
		util.Log(fmt.Sprintf("[ArticleContentProcessWorker] BEGIN Block #%d", i+1), util.LogTypeLog)
		if block.Title == nil && block.Body == nil {
			util.Log(fmt.Sprintf("[ArticleContentProcessWorker] EARLY RETURN: block #%d is nil", i+1), util.LogTypeError)
			continue
		}
		util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Block #%d: %d title sentences, %d body sentences", i+1, len(block.Title), len(block.Body)), util.LogTypeLog)
		// Title sentences
		for subIdx, s := range block.Title {
			util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Block #%d Title Sentence #%d: '%s'", i+1, subIdx, s.JP), util.LogTypeLog)
			err := saveSentenceAndWords(s, "block_title", subIdx, subIdx, block.Img, articleID)
			if err != nil {
				util.Log(fmt.Sprintf("[ArticleContentProcessWorker] ERROR saving title sentence #%d in block #%d: %v", subIdx, i+1, err), util.LogTypeError)
			} else {
				util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Saved title sentence #%d in block #%d", subIdx, i+1), util.LogTypeLog)
			}
		}
		// Body sentences
		for subIdx, s := range block.Body {
			util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Block #%d Body Sentence #%d: '%s'", i+1, subIdx, s.JP), util.LogTypeLog)
			err := saveSentenceAndWords(s, "block_body", subIdx, subIdx, block.Img, articleID)
			if err != nil {
				util.Log(fmt.Sprintf("[ArticleContentProcessWorker] ERROR saving body sentence #%d in block #%d: %v", subIdx, i+1, err), util.LogTypeError)
			} else {
				util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Saved body sentence #%d in block #%d", subIdx, i+1), util.LogTypeLog)
			}
		}
		util.Log(fmt.Sprintf("[ArticleContentProcessWorker] END Block #%d", i+1), util.LogTypeLog)
	}
	util.Log("[ArticleContentProcessWorker] EXIT: All content blocks processed.", util.LogTypeLog)
	done <- true
}

/*
func HandleArticleURL(url string, categoryId int) {
	util.Log(fmt.Sprintf("[HandleArticleURL] Handling article URL: %s", url), util.LogTypeLog)
	article, err := ArticleContentWorker(url, categoryId)
	if err != nil {
		util.Log(fmt.Sprintf("[HandleArticleURL] Error extracting article: %v", err), util.LogTypeError)
		return
	}
	done := make(chan bool)
	go ArticleContentProcessWorker(article, done)
	<-done
	util.Log("[HandleArticleURL] Article fully processed.", util.LogTypeLog)
}*/

// ProcessArticleContent processes the entire ArticleContent struct
func ProcessArticleContent(article *types.ArticleContent) {
	titleJPs := make([]string, len(article.Title))
	for i, s := range article.Title {
		titleJPs[i] = s.JP
	}
	util.Log(fmt.Sprintf("[ProcessArticleContent] Processing Article Title: '%s'", strings.Join(titleJPs, "; ")), util.LogTypeLog)
	summaryJPs := make([]string, len(article.Summary))
	for i, s := range article.Summary {
		summaryJPs[i] = s.JP
	}
	util.Log(fmt.Sprintf("[ProcessArticleContent] Processing Article Summary: '%s'", strings.Join(summaryJPs, "; ")), util.LogTypeLog)

	for i, block := range article.Content {
		util.Log(fmt.Sprintf("[ProcessArticleContent] Block #%d: %d title sentences, %d body sentences", i+1, len(block.Title), len(block.Body)), util.LogTypeLog)
	}
	util.Log("[ProcessArticleContent] All content blocks processed.", util.LogTypeLog)
}
