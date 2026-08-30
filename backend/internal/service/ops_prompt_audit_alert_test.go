package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptAuditPoolFailureEmailUsesOpsDeliveryPolicy(t *testing.T) {
	const canary = "PROMPT_AUDIT_ALERT_CANARY_DO_NOT_SEND"
	smtp := startNotificationEmailTestSMTPServer(t)
	repo := newNotificationEmailMemorySettingRepo()
	require.NoError(t, repo.SetMultiple(context.Background(), smtp.settings()))
	config, err := json.Marshal(OpsEmailNotificationConfig{Alert: OpsEmailAlertConfig{
		Enabled: true, Recipients: []string{"ops@example.com"}, MinSeverity: "critical", RateLimitPerHour: 1,
	}})
	require.NoError(t, err)
	require.NoError(t, repo.Set(context.Background(), SettingKeyOpsEmailNotificationConfig, string(config)))

	opsService := &OpsService{settingRepo: repo}
	emailService := NewEmailService(repo, nil)
	_ = NewNotificationEmailService(repo, emailService)
	evaluator := NewOpsAlertEvaluatorService(opsService, nil, emailService, nil, nil, nil)

	require.True(t, evaluator.SendPromptAuditPoolFailureEmail(context.Background(), 5, canary))
	require.False(t, evaluator.SendPromptAuditPoolFailureEmail(context.Background(), 5, "invalid"), "shared hourly limiter must suppress duplicate alert")
	require.Equal(t, int64(1), smtp.messageCount())
	body := smtp.lastMessageBody(t)
	require.Contains(t, body, "Prompt Audit guard pool consecutive failures")
	require.Contains(t, body, "unknown")
	require.NotContains(t, body, canary)
}
