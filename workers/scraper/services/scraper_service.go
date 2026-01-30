package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"scraped-worker/domain"
	"scraped-worker/repositories"

	"golang.org/x/net/html"
)

var stopWords = map[string]bool{
	"the": true, "and": true, "is": true, "in": true, "to": true, "of": true, "a": true,
}

type ScraperService struct {
	SQSClient      repositories.SQSClient
	RedisClient    repositories.RedisClient
	PageFetcher    repositories.PageFetcher
	InputQueueURL  string
	WriterQueueURL string
	ImageQueueURL  string
}

func NewScraperService(
	sqsClient repositories.SQSClient,
	redisClient repositories.RedisClient,
	pageFetcher repositories.PageFetcher,
	inputQueueURL, writerQueueURL, imageQueueURL string,
) *ScraperService {
	return &ScraperService{
		SQSClient:      sqsClient,
		RedisClient:    redisClient,
		PageFetcher:    pageFetcher,
		InputQueueURL:  inputQueueURL,
		WriterQueueURL: writerQueueURL,
		ImageQueueURL:  imageQueueURL,
	}
}

func (s *ScraperService) ProcessMessage(msg domain.ScrapeMessage) {
	log.Printf("Scraping URL: %s, Depth: %d", msg.URL, msg.Depth)

	// Ensure current URL is marked as visited (handles seed URL case)
	visitedKey := fmt.Sprintf("scrape:%d:visited", msg.ScrapingID)
	// We don't check the result here because if it's already visited, 
	// it might be a race or retry, but we proceed to extract links anyway 
	// (or should we stop? If it's already visited, we might have processed it? 
	// But let's just ensure it's in the set for its children to see).
	// Ideally, we shouldn't be here if it was visited, but SEED is exception.
	s.RedisClient.SAdd(context.TODO(), visitedKey, msg.URL)

	resp, err := s.PageFetcher.Fetch(msg.URL)
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

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.TextToken {
			text := string(tokenizer.Text())
			s.processText(text, terms)
		} else if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			token := tokenizer.Token()
			if token.Data == "a" {
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

	// Send to Writer
	writerMsg := domain.WriterMessage{
		Type:       "page_data",
		URL:        msg.URL,
		Terms:      terms,
		Links:      links,
		ScrapingID: msg.ScrapingID,
	}
	// Use background context for sending messages
	ctx := context.TODO()
	s.SQSClient.SendMessage(ctx, s.WriterQueueURL, writerMsg)

	// Send Images to Image Queue
	for _, imgURL := range images {
		imgMsg := domain.ImageMessage{
			URL:          imgURL,
			OriginalURL:  msg.URL,
			ScrapingID:   msg.ScrapingID,
		}
		s.SQSClient.SendMessage(ctx, s.ImageQueueURL, imgMsg)
	}

	// Prepare new links (with cycle detection)
	var linksToSend []string
	if msg.Depth > 0 {
		visitedKey := fmt.Sprintf("scrape:%d:visited", msg.ScrapingID)
		
		for _, link := range links {
			if strings.HasPrefix(link, "http") {
				// Cycle Detection: Atomic check-and-set
				// If SAdd returns 1, it's new. If 0, it's already visited.
				isNew, err := s.RedisClient.SAdd(ctx, visitedKey, link)
				if err != nil {
					log.Printf("error checking visited set for %s: %v", link, err)
					// If Redis fails, we might want to skip or proceed. 
					// Proceeding might cause loops, skipping might miss data.
					// Let's log and skip to be safe against loops? 
					// Or proceed to ensure coverage? 
					// Given performance concern, maybe skip. But let's proceed to allow retry?
					// Use default: treat as new (safe but risky for loops) or treat as old?
					// Let's assume treat as old (skip) to fail safe against infinite loops.
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
		key := fmt.Sprintf("scrape:%d:pending", msg.ScrapingID)
		err := s.RedisClient.IncrBy(ctx, key, int64(len(linksToSend)))
		if err != nil {
			log.Printf("CRITICAL: failed to increment redis for scraping %d: %v. Aborting to prevent race condition.", msg.ScrapingID, err)
			// Return here to let SQS redeliver this message later. 
			// If we proceed without incrementing, we risk "premature completion" signal.
			return
		}
	}

	// 2. Send SQS messages
	failedCount := 0
	for _, link := range linksToSend {
		newMsg := domain.ScrapeMessage{
			URL:          link,
			Depth:        msg.Depth - 1,
			ScrapingID:   msg.ScrapingID,
		}
		err := s.SQSClient.SendMessage(ctx, s.InputQueueURL, newMsg)
		if err != nil {
			log.Printf("failed to send message for link %s: %v", link, err)
			failedCount++
		}
	}

	// Compensate for failed sends
	if failedCount > 0 {
		key := fmt.Sprintf("scrape:%d:pending", msg.ScrapingID)
		// Subtract the failed count so the counter reflects the actual number of pending tasks in SQS
		err := s.RedisClient.IncrBy(ctx, key, int64(-failedCount)) 
		if err != nil {
			log.Printf("CRITICAL: failed to compensate redis for failed sends (scraping %d, count %d): %v", msg.ScrapingID, failedCount, err)
			// If this fails, we are in trouble (zombie job). 
			// But simpler to just log for now or maybe panic/exit to restart worker?
		}
	}

	// 3. Decrement Redis (Completion Check)
	key := fmt.Sprintf("scrape:%d:pending", msg.ScrapingID)
	val, err := s.RedisClient.Decr(ctx, key)
	if err != nil {
		log.Printf("failed to decrement redis: %v", err)
		return
	}

	// 4. Check for Completion
	if val == 0 {
		log.Printf("Scraping %d completed! Sending notification to Writer.", msg.ScrapingID)
		completionMsg := domain.WriterMessage{
			Type:         "scraping_complete",
			ScrapingID:   msg.ScrapingID,
		}
		s.SQSClient.SendMessage(ctx, s.WriterQueueURL, completionMsg)
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
