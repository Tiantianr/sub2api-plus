package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptAuditGuardNodeSnapshotMigrationIsAdditiveWithoutDataRewrite(t *testing.T) {
	raw, err := FS.ReadFile("242_prompt_audit_guard_node_snapshot.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "add column if not exists guard_endpoint_name")
	require.Contains(t, sql, "add column if not exists guard_model")
	require.NotContains(t, sql, "update prompt_audit_events")
	require.NotContains(t, sql, "base_url")
	require.NotContains(t, sql, "token")
}
