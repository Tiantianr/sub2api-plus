package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentModerationBlockingExemptMigrationIsAdditiveAndDefaultFalse(t *testing.T) {
	raw, err := FS.ReadFile("246_content_moderation_blocking_exempt.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "alter table content_moderation_logs")
	require.Contains(t, sql, "add column if not exists blocking_exempt_at_request boolean not null default false")
	require.NotContains(t, sql, "drop ")
	require.NotContains(t, sql, "update ")
}
