package migrations

import (
	"strings"
	"testing"
)

func TestPromptAuditChatSessionsMigrationDefinesBackupSafeSessionStorage(t *testing.T) {
	raw, err := FS.ReadFile("247_prompt_audit_chat_sessions.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(raw))
	for _, fragment := range []string{
		"create table if not exists prompt_audit_sessions",
		"create table if not exists prompt_audit_chat_records",
		"unique (session_id, content_hash)",
		"add column if not exists chat_record_id bigint",
		"md5('sub2api:legacy-prompt-audit-record:v1:' || e.id::text)",
		"update prompt_audit_events set full_prompt = ''",
		"delete from prompt_audit_event_contexts",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if strings.Contains(sql, "references prompt_audit_chat_records") {
		t.Fatal("event metadata must remain restorable when chat data is excluded")
	}
	if strings.Contains(sql, "chr(0)") {
		t.Fatal("PostgreSQL text expressions cannot contain NUL characters")
	}
}
