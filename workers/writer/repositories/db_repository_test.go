package repositories

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"writer-worker/domain"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	return gormDB, mock
}

func TestNewDBRepository_Default(t *testing.T) {
	repo := NewDBRepository(nil, 0)
	assert.Equal(t, 100, repo.batchSize)
}

func TestInsertPageData_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		URL:        "http://example.com",
		ScrapingID: 123,
		Links:      []string{"http://link1.com"},
	}

	// Insert Scraped Page
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "scraped_pages"`).
		WithArgs(msg.ScrapingID, msg.URL, "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	// Insert Page Links
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "page_links"`).
		WithArgs(123, 1, "http://link1.com").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	err := repo.InsertPageData(msg)
	assert.NoError(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInsertPageData_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		URL:        "http://example.com",
		ScrapingID: 123,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "scraped_pages"`).
		WillReturnError(errors.New("db error"))
	mock.ExpectRollback()

	err := repo.InsertPageData(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert scraped page")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInsertImageExplanation_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		PageURL:     "http://example.com",
		ScrapingID:  123,
		URL:         "http://img.com",
		Explanation: "desc",
		S3Path:      "path",
	}

	// Find page
	mock.ExpectQuery(`SELECT \* FROM "scraped_pages"`).
		WithArgs(msg.PageURL, msg.ScrapingID, 1). // Limit 1
		WillReturnRows(sqlmock.NewRows([]string{"id", "url", "scraping_id", "scraped_at"}).
			AddRow(1, msg.PageURL, 123, time.Now()))

	// Check if image exists (return not found to trigger insert)
	mock.ExpectQuery(`SELECT \* FROM "page_images"`).
		WithArgs(1, "path", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	// Insert Image (Upsert)
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "page_images".*`).
		WithArgs(123, 1, "http://img.com", "desc", "path").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	err := repo.InsertImageExplanation(msg)
	assert.NoError(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCompleteScraping_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	// CompleteScraping is now a no-op hook that doesn't execute SQL
	err := repo.CompleteScraping(123)
	assert.NoError(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
func TestInsertPageSummary_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		URL:        "http://example.com",
		ScrapingID: 123,
		Summary:    "This is a summary",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "scraped_pages" SET`).
		WithArgs("This is a summary", msg.URL, msg.ScrapingID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.InsertPageSummary(msg)
	assert.NoError(t, err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInsertPageSummary_Error_NoRows(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		URL:        "http://example.com",
		ScrapingID: 123,
		Summary:    "This is a summary",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "scraped_pages" SET`).
		WithArgs("This is a summary", msg.URL, msg.ScrapingID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected
	mock.ExpectCommit()

	err := repo.InsertPageSummary(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no page found to update summary")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInsertImageExplanation_Error_NoPage(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		PageURL:     "http://example.com",
		ScrapingID:  123,
		URL:         "http://img.com",
		Explanation: "desc",
	}

	// Find page - No rows found
	mock.ExpectQuery(`SELECT \* FROM "scraped_pages"`).
		WithArgs(msg.PageURL, msg.ScrapingID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	err := repo.InsertImageExplanation(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find page")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInsertPageData_Error_Links(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		URL:        "http://example.com",
		ScrapingID: 123,
		Links:      []string{"http://link1.com"},
	}

	// Insert Scraped Page Success
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "scraped_pages"`).
		WithArgs(msg.ScrapingID, msg.URL, "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	// Terms skipped because empty

	// Insert Page Links Failure
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "page_links"`).
		WithArgs(123, 1, "http://link1.com").
		WillReturnError(errors.New("links db error"))
	mock.ExpectRollback()

	err := repo.InsertPageData(msg)
	assert.NoError(t, err)
}

func TestInsertImageExplanation_InsertError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		PageURL:     "http://example.com",
		ScrapingID:  123,
		URL:         "http://img.com",
		Explanation: "desc",
		S3Path:      "path",
	}

	// Find page Success
	mock.ExpectQuery(`SELECT \* FROM "scraped_pages"`).
		WithArgs(msg.PageURL, msg.ScrapingID, 1). // Limit 1
		WillReturnRows(sqlmock.NewRows([]string{"id", "url", "scraping_id", "scraped_at"}).
			AddRow(1, msg.PageURL, 123, time.Now()))

	// Check if image exists
	mock.ExpectQuery(`SELECT \* FROM "page_images"`).
		WithArgs(1, "path", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	// Insert Image Failure
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "page_images".*`).
		WillReturnError(errors.New("image db error"))
	mock.ExpectRollback()

	err := repo.InsertImageExplanation(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert image for S3Path")
}

func TestInsertPageSummary_Error_DB(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDBRepository(db, 100)

	msg := domain.WriterMessage{
		URL:        "http://example.com",
		ScrapingID: 123,
		Summary:    "This is a summary",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "scraped_pages" SET`).
		WithArgs("This is a summary", msg.URL, msg.ScrapingID).
		WillReturnError(errors.New("db error"))
	mock.ExpectRollback()

	err := repo.InsertPageSummary(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update page summary")
}
