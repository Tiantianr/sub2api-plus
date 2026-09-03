//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"
)

func TestContentModerationRepositoryPersistsAndLoadsKeywordInputCiphertext(t *testing.T) {
	ctx := context.Background()
	repo := NewContentModerationRepository(integrationDB)
	log := &service.ContentModerationLog{
		RequestID:         "keyword-input-ciphertext-integration",
		Action:            service.ContentModerationActionKeywordBlock,
		Flagged:           true,
		HighestCategory:   "keyword",
		HighestScore:      1,
		MatchedKeyword:    "blocked",
		InputExcerpt:      "bounded excerpt",
		InputCiphertext:   "encrypted-complete-input",
		CategoryScores:    map[string]float64{"keyword": 1},
		ThresholdSnapshot: map[string]float64{},
	}

	require.NoError(t, repo.CreateLog(ctx, log))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM content_moderation_logs WHERE id = $1", log.ID)
	})

	var storedExcerpt, storedCiphertext string
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT input_excerpt, input_ciphertext FROM content_moderation_logs WHERE id = $1", log.ID,
	).Scan(&storedExcerpt, &storedCiphertext))
	require.Equal(t, "bounded excerpt", storedExcerpt)
	require.Equal(t, "encrypted-complete-input", storedCiphertext)

	detail, err := repo.GetLogInput(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, "encrypted-complete-input", detail.InputCiphertext)
}
