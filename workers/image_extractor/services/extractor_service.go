package services

import (
	"context"
	"fmt"
	"log"
	"mime"
	"path/filepath"
	"strings"

	"image-extractor-worker/domain"

	"github.com/google/uuid"
)

type SQSRepository interface {
	SendMessage(ctx context.Context, queueURL string, body interface{}) error
}

type S3Repository interface {
	UploadBytes(ctx context.Context, bucket, key string, data []byte, contentType string) (string, error)
}

type HTTPRepository interface {
	DownloadImage(url string) ([]byte, string, error)
}

type ExtractorService struct {
	sqsRepo                SQSRepository
	s3Repo                 S3Repository
	httpRepo               HTTPRepository
	writerQueueURL         string
	imageExplainerQueueURL string
	imagesBucket           string
}

func NewExtractorService(
	sqsRepo SQSRepository,
	s3Repo S3Repository,
	httpRepo HTTPRepository,
	writerQueueURL string,
	imageExplainerQueueURL string,
	imagesBucket string,
) *ExtractorService {
	return &ExtractorService{
		sqsRepo:                sqsRepo,
		s3Repo:                 s3Repo,
		httpRepo:               httpRepo,
		writerQueueURL:         writerQueueURL,
		imageExplainerQueueURL: imageExplainerQueueURL,
		imagesBucket:           imagesBucket,
	}
}

func (s *ExtractorService) ProcessMessage(ctx context.Context, msg domain.ImageMessage) error {
	log.Printf("Processing image: %s for scraping %d", msg.URL, msg.ScrapingID)

	// 1. Download image
	data, contentType, err := s.httpRepo.DownloadImage(msg.URL)
	var s3Path string
	if err != nil {
		log.Printf("Failed to download image %s: %v", msg.URL, err)
	} else {
		// 2. Upload to S3
		ext := s.getExtension(msg.URL, contentType)
		s3Key := fmt.Sprintf("%d/%s.%s", msg.ScrapingID, uuid.New().String(), ext)

		path, err := s.s3Repo.UploadBytes(ctx, s.imagesBucket, s3Key, data, contentType)
		if err != nil {
			log.Printf("Internal S3 upload failure for %s: %v", msg.URL, err)
		} else {
			s3Path = path
			log.Printf("Uploaded image to %s", s3Path)
		}
	}

	// 3. Send to Writer (Metadata)
	writerMsg := domain.WriterMessage{
		Type:        "image_explanation",
		URL:         msg.URL,
		OriginalURL: msg.OriginalURL,
		PageURL:     msg.OriginalURL,
		ScrapingID:  msg.ScrapingID,
		S3Path:      s3Path,
	}
	if err := s.sqsRepo.SendMessage(ctx, s.writerQueueURL, writerMsg); err != nil {
		return fmt.Errorf("failed to send image metadata to writer: %w", err)
	}
	log.Printf("Sent image metadata for %s to writer queue", msg.URL)

	// 4. Send to Explainer (if S3 upload succeeded)
	if s3Path != "" {
		explainerMsg := domain.ImageExtractorMessage{
			ImageURL:    msg.URL,
			OriginalURL: msg.OriginalURL,
			ScrapingID:  msg.ScrapingID,
			S3Path:      s3Path,
		}
		if err := s.sqsRepo.SendMessage(ctx, s.imageExplainerQueueURL, explainerMsg); err != nil {
			log.Printf("failed to send image to explainer queue: %v", err)
		} else {
			log.Printf("Sent image to explainer queue: %s", msg.URL)
		}
	}

	return nil
}

func (s *ExtractorService) getExtension(url, contentType string) string {
	ext := "bin"

	// Try from content type
	if contentType != "" {
		exts, err := mime.ExtensionsByType(contentType)
		if err == nil && len(exts) > 0 {
			ext = strings.TrimPrefix(exts[0], ".")
		}
	}

	// Fallback to URL extension if still bin or no content type
	if ext == "bin" || contentType == "" {
		urlExt := filepath.Ext(strings.Split(url, "?")[0])
		if urlExt != "" && len(urlExt) < 6 {
			ext = strings.TrimPrefix(urlExt, ".")
		}
	}

	return ext
}
