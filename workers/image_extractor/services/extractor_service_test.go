package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"image-extractor-worker/domain"
)

type MockSQSRepo struct {
	mock.Mock
}

func (m *MockSQSRepo) SendMessage(ctx context.Context, queueURL string, body interface{}) error {
	args := m.Called(ctx, queueURL, body)
	return args.Error(0)
}

type MockS3Repo struct {
	mock.Mock
}

func (m *MockS3Repo) UploadBytes(ctx context.Context, bucket, key string, data []byte, contentType string) (string, error) {
	args := m.Called(ctx, bucket, key, data, contentType)
	return args.String(0), args.Error(1)
}

type MockHTTPRepo struct {
	mock.Mock
}

func (m *MockHTTPRepo) DownloadImage(url string) ([]byte, string, error) {
	args := m.Called(url)
	return args.Get(0).([]byte), args.String(1), args.Error(2)
}

func TestProcessMessage_Success(t *testing.T) {
	sqs := new(MockSQSRepo)
	s3 := new(MockS3Repo)
	http := new(MockHTTPRepo)
	srv := NewExtractorService(sqs, s3, http, "writer-q", "explainer-q", "bucket")

	msg := domain.ImageMessage{
		URL:         "http://test.com/img.jpg",
		OriginalURL: "http://test.com",
		ScrapingID:  123,
	}

	http.On("DownloadImage", msg.URL).Return([]byte("data"), "image/jpeg", nil)
	s3.On("UploadBytes", mock.Anything, "bucket", mock.Anything, []byte("data"), "image/jpeg").Return("s3://bucket/key.jpg", nil)

	sqs.On("SendMessage", mock.Anything, "writer-q", mock.MatchedBy(func(m domain.WriterMessage) bool {
		return m.S3Path == "s3://bucket/key.jpg" && m.ScrapingID == 123
	})).Return(nil)

	sqs.On("SendMessage", mock.Anything, "explainer-q", mock.MatchedBy(func(m domain.ImageExtractorMessage) bool {
		return m.S3Path == "s3://bucket/key.jpg" && m.ScrapingID == 123
	})).Return(nil)

	err := srv.ProcessMessage(context.Background(), msg)
	assert.NoError(t, err)

	http.AssertExpectations(t)
	s3.AssertExpectations(t)
	sqs.AssertExpectations(t)
}

func TestProcessMessage_DownloadError(t *testing.T) {
	sqs := new(MockSQSRepo)
	s3 := new(MockS3Repo)
	http := new(MockHTTPRepo)
	srv := NewExtractorService(sqs, s3, http, "writer-q", "explainer-q", "bucket")

	msg := domain.ImageMessage{
		URL:        "http://test.com/img.jpg",
		ScrapingID: 123,
	}

	http.On("DownloadImage", msg.URL).Return([]byte(nil), "", assert.AnError)

	// Should still send metadata to writer (but with empty s3_path)
	sqs.On("SendMessage", mock.Anything, "writer-q", mock.MatchedBy(func(m domain.WriterMessage) bool {
		return m.S3Path == "" && m.ScrapingID == 123
	})).Return(nil)

	err := srv.ProcessMessage(context.Background(), msg)
	assert.NoError(t, err)
}

func TestGetExtension(t *testing.T) {
	srv := &ExtractorService{}

	ext := srv.getExtension("http://test.com/a.jpg", "image/jpeg")
	assert.True(t, ext == "jpeg" || ext == "jpe")
	assert.Equal(t, "png", srv.getExtension("http://test.com/a.png", "image/png"))
	assert.Equal(t, "jpg", srv.getExtension("http://test.com/a.jpg?query=1", ""))
	assert.Equal(t, "bin", srv.getExtension("http://test.com/a", ""))
}
