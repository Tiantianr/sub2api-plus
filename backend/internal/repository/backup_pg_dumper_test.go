package repository

import (
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPgDumpArgsExcludePromptAuditChatContent(t *testing.T) {
	args := pgDumpArgs(&config.DatabaseConfig{Host: "db", Port: 5432, User: "user", DBName: "app"})
	require.Contains(t, args, "--exclude-table-data=prompt_audit_chat_records")
	require.Contains(t, args, "--exclude-table-data=prompt_audit_event_contexts")
}
