package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptAuditFailureEventsMigrationBackfillsLightweightEvents(t *testing.T) {
	raw, err := FS.ReadFile("243_prompt_audit_failure_events.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))

	for _, field := range []string{"error_code", "error_message"} {
		require.Contains(t, sql, "add column if not exists "+field)
	}
	for _, field := range []string{"failure_kind", "failure_disposition", "failure_allowed", "failed_chunk_index"} {
		require.NotContains(t, sql, "add column if not exists "+field)
	}
	require.Contains(t, sql, "decision in ('pass', 'flag', 'critical', 'failed')")
	require.Contains(t, sql, "risk_level in ('low', 'medium', 'high', 'critical', 'unknown')")
	require.Contains(t, sql, "action in ('allow', 'warn', 'block', 'error')")
	require.Contains(t, sql, "from prompt_audit_jobs as j")
	require.Contains(t, sql, "where j.status = 'failed'")
	require.Contains(t, sql, "not exists (select 1 from prompt_audit_events")
	require.NotContains(t, sql, "raw_guard")
	require.NotContains(t, sql, "base_url")
	require.NotContains(t, sql, "token_ciphertext")
}
