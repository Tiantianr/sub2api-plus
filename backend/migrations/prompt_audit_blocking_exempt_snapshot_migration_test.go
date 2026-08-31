package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptAuditBlockingExemptSnapshotMigrationAddsImmutableMarkers(t *testing.T) {
	raw, err := FS.ReadFile("244_prompt_audit_blocking_exempt_snapshot.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "alter table prompt_audit_jobs")
	require.Contains(t, sql, "alter table prompt_audit_events")
	require.Equal(t, 2, strings.Count(sql, "blocking_exempt_at_request"))
	require.Equal(t, 2, strings.Count(sql, "add column if not exists"))
	require.Equal(t, 2, strings.Count(sql, "boolean not null default false"))
	require.NotContains(t, sql, "update ")
}
