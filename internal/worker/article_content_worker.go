package worker

import (
	"fmt"
	"net/url"
	"nhknewseasybkend/internal/model"
	"nhknewseasybkend/internal/util"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Sentence represents a sentence and its tokenized words
type Sentence struct {
	Text  string
	Words []model.AITokenizedWord
}

type ContentBlock struct {
	Title []Sentence
	Img   string
	Body  []Sentence
}

type ArticleContent struct {
	Title   []Sentence
	Video   string
	Date    string
	Summary []Sentence
	Content []ContentBlock
}

// SaveArticleToDatabase saves the article content and unique words to the database
func SaveArticleToDatabase(content *ArticleContent, uniqueWords []model.AITokenizedWord) error {
	// Use model.AITokenizedWord for uniqueWords
	util.Log("[SaveArticleToDatabase] Saving article and unique words to database...", util.LogTypeLog)
	// TODO: Implement actual database save logic here
	util.Log(fmt.Sprintf("[SaveArticleToDatabase] Article title: %v", content.Title), util.LogTypeLog)
	util.Log(fmt.Sprintf("[SaveArticleToDatabase] Number of unique words: %d", len(uniqueWords)), util.LogTypeLog)
	// Return nil for now
	return nil
}

// ExtractUniqueWordsFromArticleContent returns a slice of unique AITokenizedWord from the entire ArticleContent
func ExtractUniqueWordsFromArticleContent(article *ArticleContent) []model.AITokenizedWord {
	unique := make(map[string]model.AITokenizedWord)
	// Helper to process a slice of Sentence
	processSentences := func(sentences []Sentence) {
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
	result := make([]model.AITokenizedWord, 0, len(unique))
	for _, w := range unique {
		result = append(result, w)
	}
	return result
}

// ArticleContentWorker runs periodically to process article URLs
// and extract Japanese sentences/content
// You can expand this to accept a list of URLs or integrate with your DB
func ArticleContentWorker(url string) (*ArticleContent, error) {
	util.Log(fmt.Sprintf("[ArticleContentWorker] Fetching article from URL: %s", url), util.LogTypeLog)
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

	// Extract unique words before saving to database
	uniqueWords := ExtractUniqueWordsFromArticleContent(content)
	util.Log(fmt.Sprintf("[ArticleContentWorker] Found %d unique words for article %s", len(uniqueWords), url), util.LogTypeLog)

	// Save to database
	errSave := SaveArticleToDatabase(content, uniqueWords)
	if errSave != nil {
		util.Log(fmt.Sprintf("[ArticleContentWorker] Error saving to database: %v", errSave), util.LogTypeError)
	}

	return content, nil
}

// parseArticleHTML parses the HTML and extracts article content
func parseArticleHTML(html, baseURL string) (*ArticleContent, error) {
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

	var contentBlocks []ContentBlock

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
			contentBlocks = append(contentBlocks, ContentBlock{
				Title: blockTitleSentences,
				Img:   img,
				Body:  bodySentences,
			})
			blockCount++
			util.Log(fmt.Sprintf("[ArticleContentWorker] Extracted block #%d: title='%s', img='%s', body length=%d", blockCount, blockTitleRaw, img, len(bodyRaw)), util.LogTypeLog)
		}
	})

	util.Log(fmt.Sprintf("[ArticleContentWorker] Total content blocks extracted: %d", blockCount), util.LogTypeLog)

	return &ArticleContent{
		Title:   titleSentences,
		Video:   video,
		Date:    date,
		Summary: summarySentences,
		Content: contentBlocks,
	}, nil
}

// Updated processContent to call ArticleSentenceProcessWorker directly
func processContent(title, body string) ([]Sentence, error) {
	util.Log(fmt.Sprintf("[processContent] Processing block: Title='%s', Body length=%d", title, len(body)), util.LogTypeLog)
	results, err := ArticleSentenceProcessWorker(title, body)
	util.Log(fmt.Sprintf("[processContent] Finished processing block: Title='%s'", title), util.LogTypeLog)
	sentences := make([]Sentence, 0, len(results))
	for _, r := range results {
		sentences = append(sentences, Sentence{Text: r.Sentence, Words: r.Words})
	}
	return sentences, err
}

// ArticleContentProcessWorker processes the full ArticleContent struct
func ArticleContentProcessWorker(article *ArticleContent, done chan<- bool) {
	util.Log("[ArticleContentProcessWorker] Starting processing of ArticleContent...", util.LogTypeLog)
	util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Title: '%s'", article.Title), util.LogTypeLog)
	util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Summary: '%s'", article.Summary), util.LogTypeLog)

	for i, block := range article.Content {
		util.Log(fmt.Sprintf("[ArticleContentProcessWorker] Block #%d: %d title sentences, %d body sentences", i+1, len(block.Title), len(block.Body)), util.LogTypeLog)
	}
	util.Log("[ArticleContentProcessWorker] All content blocks processed.", util.LogTypeLog)
	done <- true
}

func HandleArticleURL(url string) {
	util.Log(fmt.Sprintf("[HandleArticleURL] Handling article URL: %s", url), util.LogTypeLog)
	article, err := ArticleContentWorker(url)
	if err != nil {
		util.Log(fmt.Sprintf("[HandleArticleURL] Error extracting article: %v", err), util.LogTypeError)
		return
	}
	done := make(chan bool)
	go ArticleContentProcessWorker(article, done)
	<-done
	util.Log("[HandleArticleURL] Article fully processed.", util.LogTypeLog)
}

// ProcessArticleContent processes the entire ArticleContent struct
func ProcessArticleContent(article *ArticleContent) {
	util.Log(fmt.Sprintf("[ProcessArticleContent] Processing Article Title: '%s'", article.Title), util.LogTypeLog)
	util.Log(fmt.Sprintf("[ProcessArticleContent] Processing Article Summary: '%s'", article.Summary), util.LogTypeLog)

	for i, block := range article.Content {
		util.Log(fmt.Sprintf("[ProcessArticleContent] Block #%d: %d title sentences, %d body sentences", i+1, len(block.Title), len(block.Body)), util.LogTypeLog)
	}
	util.Log("[ProcessArticleContent] All content blocks processed.", util.LogTypeLog)
}
