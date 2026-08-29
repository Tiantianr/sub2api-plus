package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptAuditDeepReviewMigrationAddsModeConstraintsAndIndex(t *testing.T) {
	raw, err := FS.ReadFile("239_prompt_audit_deep_review.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "'async_deep'")
	require.Contains(t, sql, "chk_prompt_audit_jobs_execution_mode")
	require.Contains(t, sql, "chk_prompt_audit_events_observability")
	require.Contains(t, sql, "not valid")
	require.Contains(t, sql, "validate constraint")

	indexRaw, err := FS.ReadFile("240_prompt_audit_events_mode_index_notx.sql")
	require.NoError(t, err)
	indexSQL := strings.ToLower(string(indexRaw))
	require.Contains(t, indexSQL, "create index concurrently if not exists")
	require.Contains(t, indexSQL, "idx_prompt_audit_events_mode_created")
}
