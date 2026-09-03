package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentModerationKeywordInputCiphertextMigrationIsAdditiveAndEncryptedOnly(t *testing.T) {
	raw, err := FS.ReadFile("245_content_moderation_keyword_input_ciphertext.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "alter table content_moderation_logs")
	require.Contains(t, sql, "add column if not exists input_ciphertext text not null default ''")
	require.NotContains(t, sql, "input_content")
	require.NotContains(t, sql, "update ")
}
