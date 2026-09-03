package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type contentModerationHandlerSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

type contentModerationHandlerLogRepo struct {
	input *service.ContentModerationLogInput
	err   error
}

func (r *contentModerationHandlerLogRepo) CreateLog(context.Context, *service.ContentModerationLog) error {
	return nil
}

func (r *contentModerationHandlerLogRepo) ListLogs(context.Context, service.ContentModerationLogFilter) ([]service.ContentModerationLog, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{}, nil
}

func (r *contentModerationHandlerLogRepo) GetLogInput(context.Context, int64) (*service.ContentModerationLogInput, error) {
	return r.input, r.err
}

func (r *contentModerationHandlerLogRepo) CountFlaggedByUserSince(context.Context, int64, time.Time, bool) (int, error) {
	return 0, nil
}

func (r *contentModerationHandlerLogRepo) CleanupExpiredLogs(context.Context, time.Time, time.Time) (*service.ContentModerationCleanupResult, error) {
	return &service.ContentModerationCleanupResult{}, nil
}

func (r *contentModerationHandlerLogRepo) UpdateLogEmailSent(context.Context, int64, bool) error {
	return nil
}

type contentModerationHandlerEncryptor struct{}

func (contentModerationHandlerEncryptor) Encrypt(plaintext string) (string, error) {
	return "cipher:" + plaintext, nil
}

func (contentModerationHandlerEncryptor) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "cipher:"), nil
}

func (r *contentModerationHandlerSettingRepo) Get(_ context.Context, key string) (*service.Setting, error) {
	value, err := r.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return &service.Setting{Key: key, Value: value}, nil
}
func (r *contentModerationHandlerSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}
func (r *contentModerationHandlerSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}
func (r *contentModerationHandlerSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, err := r.GetValue(ctx, key); err == nil {
			out[key] = value
		}
	}
	return out, nil
}
func (r *contentModerationHandlerSettingRepo) SetMultiple(ctx context.Context, values map[string]string) error {
	for key, value := range values {
		if err := r.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}
func (r *contentModerationHandlerSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}
func (r *contentModerationHandlerSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

func TestContentModerationHandlerPersistsTextAPIMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := &contentModerationHandlerSettingRepo{values: map[string]string{}}
	svc := service.NewContentModerationService(settings, nil, nil, nil, nil, nil, nil, nil)
	handler := NewContentModerationHandler(svc)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/risk-control/config", strings.NewReader(`{"text_api_mode":"off"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateConfig(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	raw, err := settings.GetValue(context.Background(), service.SettingKeyContentModerationConfig)
	require.NoError(t, err)
	var persisted service.ContentModerationConfig
	require.NoError(t, json.Unmarshal([]byte(raw), &persisted))
	require.Equal(t, service.ContentModerationTextAPIModeOff, persisted.TextAPIMode)
	require.Contains(t, recorder.Body.String(), `"text_api_mode":"off"`)
}

func TestContentModerationHandlerGetLogInputReturnsDecryptedPublicFieldsOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &contentModerationHandlerLogRepo{input: &service.ContentModerationLogInput{
		ID:              42,
		Action:          service.ContentModerationActionKeywordBlock,
		MatchedKeyword:  "blocked",
		InputExcerpt:    "excerpt",
		InputCiphertext: "cipher:sub2api:content-moderation-keyword-input:v1:complete audited content",
	}}
	svc := service.NewContentModerationService(nil, repo, nil, nil, nil, nil, nil, nil)
	svc.SetSecretEncryptor(contentModerationHandlerEncryptor{})
	handler := NewContentModerationHandler(svc)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/risk-control/logs/42/input", nil)

	handler.GetLogInput(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "no-store, private, max-age=0", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Contains(t, recorder.Body.String(), `"content":"complete audited content"`)
	require.Contains(t, recorder.Body.String(), `"complete":true`)
	require.NotContains(t, recorder.Body.String(), "InputCiphertext")
	require.NotContains(t, recorder.Body.String(), "cipher:")
}

func TestContentModerationHandlerGetLogInputRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewContentModerationHandler(service.NewContentModerationService(nil, nil, nil, nil, nil, nil, nil, nil))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/risk-control/logs/invalid/input", nil)

	handler.GetLogInput(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
