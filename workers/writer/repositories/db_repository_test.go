package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"workers/writer/domain"
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

func TestDBRepository_InsertPageData(t *testing.T) {
	tests := []struct {
		name      string
		msg       domain.WriterMessage
		mockFunc  func(sqlmock.Sqlmock)
		wantErr   bool
		errString string
	}{
		{
			name: "Success",
			msg:  domain.WriterMessage{URL: "http://site.com", ScrapingID: 123, Links: []string{"l1"}},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "scraped_pages"`).
					WithArgs(123, "http://site.com", "").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "page_links"`).
					WithArgs(123, 1, "l1").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
		},
		{
			name: "Scraped Page Error",
			msg:  domain.WriterMessage{URL: "http://err.com", ScrapingID: 123},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "scraped_pages"`).WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			wantErr:   true,
			errString: "failed to insert scraped page",
		},
		{
			name: "Links Error (Swallowed)",
			msg:  domain.WriterMessage{URL: "http://site.com", ScrapingID: 123, Links: []string{"l1"}},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "scraped_pages"`).
					WithArgs(123, "http://site.com", "").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()

				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "page_links"`).WillReturnError(errors.New("links error"))
				mock.ExpectRollback()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			repo := NewDBRepository(db, 100)
			tt.mockFunc(mock)

			err := repo.InsertPageData(context.Background(), tt.msg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBRepository_InsertImageExplanation(t *testing.T) {
	tests := []struct {
		name      string
		msg       domain.WriterMessage
		mockFunc  func(sqlmock.Sqlmock)
		wantErr   bool
		errString string
	}{
		{
			name: "Success",
			msg:  domain.WriterMessage{PageURL: "http://site.com", ScrapingID: 123, URL: "http://img.com", Explanation: "desc", S3Path: "path"},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "scraped_pages"`).WithArgs("http://site.com", 123, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery(`SELECT \* FROM "page_images"`).WithArgs(1, "path", 1).WillReturnError(gorm.ErrRecordNotFound)
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "page_images"`).WithArgs(123, 1, "http://img.com", "desc", "path").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
		},
		{
			name: "Page Not Found",
			msg:  domain.WriterMessage{PageURL: "http://missing.com", ScrapingID: 123},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "scraped_pages"`).WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr:   true,
			errString: "failed to find page",
		},
		{
			name: "Insert Error",
			msg:  domain.WriterMessage{PageURL: "http://site.com", ScrapingID: 123, S3Path: "path"},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "scraped_pages"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery(`SELECT \* FROM "page_images"`).WillReturnError(gorm.ErrRecordNotFound)
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "page_images"`).WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			wantErr:   true,
			errString: "failed to insert image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			repo := NewDBRepository(db, 100)
			tt.mockFunc(mock)

			err := repo.InsertImageExplanation(context.Background(), tt.msg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBRepository_InsertPageSummary(t *testing.T) {
	tests := []struct {
		name      string
		msg       domain.WriterMessage
		mockFunc  func(sqlmock.Sqlmock)
		wantErr   bool
		errString string
	}{
		{
			name: "Success",
			msg:  domain.WriterMessage{URL: "http://site.com", ScrapingID: 123, Summary: "summary"},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE "scraped_pages" SET`).WithArgs("summary", "http://site.com", 123).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "No Rows Affected",
			msg:  domain.WriterMessage{URL: "http://site.com", ScrapingID: 123, Summary: "summary"},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE "scraped_pages" SET`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr:   true,
			errString: "no page found to update summary",
		},
		{
			name: "DB Error",
			msg:  domain.WriterMessage{URL: "http://site.com", ScrapingID: 123},
			mockFunc: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE "scraped_pages" SET`).WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			wantErr:   true,
			errString: "failed to update page summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			repo := NewDBRepository(db, 100)
			tt.mockFunc(mock)

			err := repo.InsertPageSummary(context.Background(), tt.msg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBRepository_CompleteScraping(t *testing.T) {
	db, _ := newMockDB(t)
	repo := NewDBRepository(db, 100)
	err := repo.CompleteScraping(context.Background(), 123)
	assert.NoError(t, err)
}
