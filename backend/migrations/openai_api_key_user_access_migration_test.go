package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIAPIKeyUserAccessMigrationExtendsExistingTriggers(t *testing.T) {
	raw, err := FS.ReadFile("248_openai_api_key_user_access.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(raw))
	require.Contains(t, sql, "a.type in ('oauth', 'apikey')")
	require.Contains(t, sql, "new.type not in ('oauth', 'apikey')")
	require.Contains(t, sql, "create or replace function grant_default_openai_oauth_accounts_to_new_user")
	require.Contains(t, sql, "create or replace function clear_invalid_openai_oauth_account_access_policy")
	require.NotContains(t, sql, "drop ")
	require.NotContains(t, sql, "delete from accounts")
}
