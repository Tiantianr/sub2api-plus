package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type contentModerationHandlerSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
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
