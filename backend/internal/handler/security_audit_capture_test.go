package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type capturePromptConfigStore struct{ cfg securityaudit.ActiveConfig }

func (s *capturePromptConfigStore) Start(context.Context) error    { return nil }
func (s *capturePromptConfigStore) Shutdown(context.Context) error { return nil }
func (s *capturePromptConfigStore) Active() (securityaudit.ActiveConfig, bool) {
	return s.cfg, true
}
func (s *capturePromptConfigStore) EffectiveMode() securityaudit.Mode { return s.cfg.EffectiveMode() }
func (s *capturePromptConfigStore) BlockingActivationDegraded() bool  { return false }
func (s *capturePromptConfigStore) Public() (securityaudit.PublicConfig, error) {
	return securityaudit.PublicConfig{}, nil
}
func (s *capturePromptConfigStore) Save(context.Context, securityaudit.UpdateConfigRequest, int64) (securityaudit.PublicConfig, error) {
	return securityaudit.PublicConfig{}, nil
}
func (s *capturePromptConfigStore) RuntimeState() (int64, int64, *time.Time, string) {
	return s.cfg.ConfigVersion, s.cfg.ConfigVersion, nil, ""
}
func (s *capturePromptConfigStore) Encrypt(value string) (string, error) { return value, nil }
func (s *capturePromptConfigStore) Decrypt(value string) (string, error) { return value, nil }

func TestSecurityAuditResponseCaptureMiddlewareCommitsFinalSSEOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	guard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Safety: Safe\nCategories: None"}}]}`))
	}))
	defer guard.Close()

	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	config := &capturePromptConfigStore{cfg: securityaudit.ActiveConfig{
		RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true, ConfigVersion: 3,
		Scanners: securityaudit.AllScannerIDs,
		Endpoints: []securityaudit.ActiveEndpoint{{
			ID: "guard", Enabled: true, BaseURL: guard.URL, Model: "guard-test", TimeoutMS: 3000, InputLimit: securityaudit.MaxInputLimit,
		}},
	}}
	prompt := securityaudit.NewPromptService(
		config, nil, securityaudit.NewRedisPayloadStore(redisClient),
		securityaudit.NewOpenAICompatibleScanner(), securityaudit.NewAtomicMetrics(),
	)
	conversationKey := securityaudit.ConversationKey(41, "chat-session")
	firstBody := []byte(`{"messages":[{"role":"user","content":"first user"}]}`)

	router := gin.New()
	router.Use(SecurityAuditResponseCaptureMiddleware())
	router.GET("/capture", func(c *gin.Context) {
		promptDecision, err := prompt.Evaluate(c.Request.Context(), securityaudit.Request{
			APIKeyID: 41, Protocol: "openai_chat_completions", Model: "gpt-test",
			ConversationKey: conversationKey, Body: firstBody,
		})
		require.NoError(t, err)
		attachSecurityAuditConversationCapture(c, &securityaudit.Decision{
			Kind: securityaudit.DecisionAllow, Prompt: promptDecision, AllowNextStage: true,
		})
		c.Header("Content-Type", "text/event-stream")
		_, _ = c.Writer.WriteString("data: {\"choices\":[{\"delta\":{\"content\":\"assistant output\"}}]}\n\n")
		_, _ = c.Writer.WriteString("data: [DONE]\n\n")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/capture", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	continued, err := prompt.Evaluate(context.Background(), securityaudit.Request{
		APIKeyID: 41, Protocol: "openai_chat_completions", Model: "gpt-test", ConversationKey: conversationKey,
		Body: []byte(`{"messages":[{"role":"user","content":"first user"},{"role":"assistant","content":"assistant output"},{"role":"user","content":"second user"}]}`),
	})
	require.NoError(t, err)
	require.Equal(t, "incremental", continued.ConversationMode)
	require.NotNil(t, continued.Capture)
	continued.Capture.Abort()
}
