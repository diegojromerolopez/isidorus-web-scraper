package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"scraped-worker/domain"

	"golang.org/x/net/html"
)

// Consumer-side interfaces
type SQSClient interface {
	SendMessage(ctx context.Context, queueURL string, messageBody interface{}) error
	DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error
}

type RedisClient interface {
	SAdd(ctx context.Context, key string, member ...interface{}) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) error
	Decr(ctx context.Context, key string) (int64, error)
}

type PageFetcher interface {
	Fetch(url string) (*http.Response, error)
}

var stopWords = map[string]bool{
	"the": true, "and": true, "is": true, "in": true, "to": true, "of": true, "a": true,
}

type ScraperService struct {
	sqsClient             SQSClient
	redisClient           RedisClient
	pageFetcher           PageFetcher
	inputQueueURL         string
	writerQueueURL        string
	imageQueueURL         string
	summarizerQueueURL    string
	imageExplainerEnabled bool
	pageSummarizerEnabled bool
}

// Functional Options Pattern
type ScraperOption func(*ScraperService)

func WithSQSClient(c SQSClient) ScraperOption {
	return func(s *ScraperService) { s.sqsClient = c }
}

func WithRedisClient(c RedisClient) ScraperOption {
	return func(s *ScraperService) { s.redisClient = c }
}

func WithPageFetcher(c PageFetcher) ScraperOption {
	return func(s *ScraperService) { s.pageFetcher = c }
}

func WithQueues(input, writer, image, summarizer string) ScraperOption {
	return func(s *ScraperService) {
		s.inputQueueURL = input
		s.writerQueueURL = writer
		s.imageQueueURL = image
		s.summarizerQueueURL = summarizer
	}
}

func WithFeatureFlags(imageExplainer, pageSummarizer bool) ScraperOption {
	return func(s *ScraperService) {
		s.imageExplainerEnabled = imageExplainer
		s.pageSummarizerEnabled = pageSummarizer
	}
}

func NewScraperService(opts ...ScraperOption) *ScraperService {
	s := &ScraperService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ScraperService) ProcessMessage(msg domain.ScrapeMessage) {
	log.Printf("Scraping URL: %s, Depth: %d", msg.URL, msg.Depth)

	// Context for I/O operations
	ctx := context.TODO()

	// Ensure current URL is marked as visited (handles seed URL case)
	visitedKey := fmt.Sprintf(domain.RedisKeyVisited, msg.ScrapingID)
	_, _ = s.redisClient.SAdd(ctx, visitedKey, msg.URL)

	pKey := fmt.Sprintf(domain.RedisKeyPending, msg.ScrapingID)
	defer func() {
		val, err := s.redisClient.Decr(ctx, pKey)
		if err != nil {
			log.Printf("failed to decrement redis: %v", err)
			return
		}

		// 4. Check for Completion
		if val == 0 {
			log.Printf("Scraping %d completed! Sending notification to Writer.", msg.ScrapingID)
			completionMsg := domain.WriterMessage{
				Type:       domain.MsgTypeScrapingComplete,
				ScrapingID: msg.ScrapingID,
			}
			if err := s.sqsClient.SendMessage(ctx, s.writerQueueURL, completionMsg); err != nil {
				log.Printf("failed to send completion signal: %v", err)
			}
		}
	}()

	resp, err := s.pageFetcher.Fetch(msg.URL)
	if err != nil {
		log.Printf("failed to fetch URL %s: %v", msg.URL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("non-200 status code for URL %s: %d", msg.URL, resp.StatusCode)
		return
	}

	tokenizer := html.NewTokenizer(resp.Body)
	terms := make(map[string]int)
	links := []string{}
	images := []string{}

	// Text accumulation for summarization
	var fullTextBuilder strings.Builder

	inScript := false
	inStyle := false

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.TextToken {
			if !inScript && !inStyle {
				text := string(tokenizer.Text())
				s.processText(text, terms)

				// Accumulate text for summarizer (simple append for now)
				// Limit size to avoid memory issues (e.g. 100KB)
				if fullTextBuilder.Len() < 100000 {
					fullTextBuilder.WriteString(text)
					fullTextBuilder.WriteString(" ")
				}
			}
		} else if tt == html.StartTagToken {
			token := tokenizer.Token()
			if token.Data == "script" {
				inScript = true
			} else if token.Data == "style" {
				inStyle = true
			} else if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			} else if token.Data == "img" {
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						images = append(images, attr.Val)
					}
				}
			}
		} else if tt == html.EndTagToken {
			token := tokenizer.Token()
			if token.Data == "script" {
				inScript = false
			} else if token.Data == "style" {
				inStyle = false
			}
		} else if tt == html.SelfClosingTagToken {
			token := tokenizer.Token()
			if token.Data == "script" {
				// Self-closing script tag (rare but possible in XHTML)
				// effectively enters and leaves, or just stays false if we handle it like this
				// But strictly if it's self-closing it has no content.
			} else if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			} else if token.Data == "img" {
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						images = append(images, attr.Val)
					}
				}
			}
		}
	}

	// Send to Writer (Page Data)
	// Priority 1: Ensure page exists in DB before sending tasks that reference it (Images, Summaries)
	writerMsg := domain.WriterMessage{
		Type:       domain.MsgTypePageData,
		URL:        msg.URL,
		Terms:      terms,
		Links:      links,
		ScrapingID: msg.ScrapingID,
	}
	if err := s.sqsClient.SendMessage(ctx, s.writerQueueURL, writerMsg); err != nil {
		log.Printf("failed to send page data to writer: %v", err)
	}

	// Send to Summarizer Queue (if configured, enabled, and populated)
	log.Printf("Checking summarizer: enabled=%v, queue=%s, textLen=%d", s.pageSummarizerEnabled, s.summarizerQueueURL, fullTextBuilder.Len())
	if s.pageSummarizerEnabled && s.summarizerQueueURL != "" && fullTextBuilder.Len() > 0 {
		summaryMsg := domain.PageSummaryMessage{
			URL:        msg.URL,
			Content:    fullTextBuilder.String(),
			ScrapingID: msg.ScrapingID,
		}
		if err := s.sqsClient.SendMessage(ctx, s.summarizerQueueURL, summaryMsg); err != nil {
			log.Printf("failed to send page summary request: %v", err)
		}
	}

	// Send Images to Image Queue (if enabled)
	if s.imageExplainerEnabled {
		for _, imgURL := range images {
			imgMsg := domain.ImageMessage{
				URL:         imgURL,
				OriginalURL: msg.URL,
				ScrapingID:  msg.ScrapingID,
			}
			if err := s.sqsClient.SendMessage(ctx, s.imageQueueURL, imgMsg); err != nil {
				log.Printf("failed to send image to image queue: %v", err)
			}
		}
	}

	// Prepare new links (with cycle detection)
	var linksToSend []string
	if msg.Depth > 0 {
		vKey := fmt.Sprintf(domain.RedisKeyVisited, msg.ScrapingID)

		for _, link := range links {
			if strings.HasPrefix(link, "http") {
				// Cycle Detection: Atomic check-and-set
				isNew, err := s.redisClient.SAdd(ctx, vKey, link)
				if err != nil {
					log.Printf("error checking visited set for %s: %v", link, err)
					continue
				}

				if isNew > 0 {
					linksToSend = append(linksToSend, link)
				}
			}
		}
	}

	// 1. Increment Redis if we have links to send
	if len(linksToSend) > 0 {
		pKey := fmt.Sprintf(domain.RedisKeyPending, msg.ScrapingID)
		err := s.redisClient.IncrBy(ctx, pKey, int64(len(linksToSend)))
		if err != nil {
			log.Printf("CRITICAL: failed to increment redis for scraping %d: %v. Aborting to prevent race condition.", msg.ScrapingID, err)
			return
		}
	}

	// 2. Send SQS messages
	failedCount := 0
	for _, link := range linksToSend {
		newMsg := domain.ScrapeMessage{
			URL:        link,
			Depth:      msg.Depth - 1,
			ScrapingID: msg.ScrapingID,
		}
		err := s.sqsClient.SendMessage(ctx, s.inputQueueURL, newMsg)
		if err != nil {
			log.Printf("failed to send message for link %s: %v", link, err)
			failedCount++
		}
	}

	// Compensate for failed sends
	if failedCount > 0 {
		pKey := fmt.Sprintf(domain.RedisKeyPending, msg.ScrapingID)
		err := s.redisClient.IncrBy(ctx, pKey, int64(-failedCount))
		if err != nil {
			log.Printf("CRITICAL: failed to compensate redis for failed sends (scraping %d, count %d): %v", msg.ScrapingID, failedCount, err)
		}
	}
}

func (s *ScraperService) processText(text string, terms map[string]int) {
	words := strings.Fields(text)
	for _, word := range words {
		word = strings.ToLower(word)
		word = strings.Trim(word, ".,!?:;\"'()")
		if len(word) > 2 && !stopWords[word] {
			terms[word]++
		}
	}
}
