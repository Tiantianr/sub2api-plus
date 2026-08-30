package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptAuditPassRetentionMigrationRemovesLegacyGlobalSwitch(t *testing.T) {
	raw, err := FS.ReadFile("241_prompt_audit_remove_global_pass_retention.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "where key = 'prompt_audit_config'")
	require.Contains(t, sql, "value::jsonb - 'store_pass_events'")
	require.NotContains(t, sql, "delete from settings")
}
