package repository

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildContentModerationLogWhere_BlockedIncludesAllBlockActions(t *testing.T) {
	where, args := buildContentModerationLogWhere(service.ContentModerationLogFilter{Result: "blocked"})

	require.Empty(t, args)
	sql := strings.Join(where, " AND ")
	require.Contains(t, sql, "l.action IN ('block', 'keyword_block', 'hash_block')")
	require.NotContains(t, sql, "l.action = 'block'")
}

func TestContentModerationListColumnsExcludeCompleteInputCiphertext(t *testing.T) {
	require.Contains(t, contentModerationListSelectColumns, "l.input_excerpt")
	require.Contains(t, contentModerationListSelectColumns, "l.blocking_exempt_at_request")
	require.NotContains(t, contentModerationListSelectColumns, "input_ciphertext")
}

func TestContentModerationRepositoryGetLogInputLoadsCiphertextOnlyByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, action, matched_keyword, input_excerpt, input_ciphertext
FROM content_moderation_logs
WHERE id = $1`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "action", "matched_keyword", "input_excerpt", "input_ciphertext"}).
			AddRow(42, service.ContentModerationActionKeywordBlock, "blocked", "excerpt", "ciphertext"))

	result, err := repo.GetLogInput(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, int64(42), result.ID)
	require.Equal(t, "ciphertext", result.InputCiphertext)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryGetLogInputMapsMissingRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	mock.ExpectQuery("SELECT id, action, matched_keyword, input_excerpt, input_ciphertext").
		WithArgs(int64(404)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetLogInput(context.Background(), 404)
	require.ErrorIs(t, err, service.ErrContentModerationLogNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryCreateLogPersistsCiphertextColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	mock.ExpectQuery("(?s)INSERT INTO content_moderation_logs.*matched_keyword, input_ciphertext.*RETURNING id, created_at").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(42, time.Now()))
	log := &service.ContentModerationLog{
		Action:            service.ContentModerationActionKeywordBlock,
		MatchedKeyword:    "blocked",
		InputExcerpt:      "excerpt",
		InputCiphertext:   "encrypted-complete-input",
		CategoryScores:    map[string]float64{"keyword": 1},
		ThresholdSnapshot: map[string]float64{},
	}

	err = repo.CreateLog(context.Background(), log)
	require.NoError(t, err)
	require.Equal(t, int64(42), log.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesHashAndShadow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND action <> 'hash_block'\n  AND action <> 'shadow'")).
		WithArgs(int64(1001), since, false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, false)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesCyberPolicyWhenRequested(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND ($3::bool IS FALSE OR action <> 'cyber_policy')")).
		WithArgs(int64(1001), since, true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, true)

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, mock.ExpectationsWereMet())
}
