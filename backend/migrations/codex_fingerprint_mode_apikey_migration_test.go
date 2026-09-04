package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration264AllowsAndBackfillsCodexFingerprintModeAPIKey(t *testing.T) {
	content, err := FS.ReadFile("264_allow_codex_fingerprint_mode_apikey.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "NOT IN ('oauth', 'apikey')")
	require.Contains(t, sql, "ELSE 'off'")
	require.Contains(t, sql, "type = 'apikey'")
	require.Contains(t, sql, "codex_fingerprint_mode")
	require.Contains(t, sql, "CREATE TRIGGER")
	require.Contains(t, sql, "must be one of off, device, session, full")
}
