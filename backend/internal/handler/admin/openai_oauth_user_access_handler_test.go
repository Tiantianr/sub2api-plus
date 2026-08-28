package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIOAuthAccessHandlerRepo struct {
	accounts []service.OpenAIOAuthAccessAccount
	applied  []service.OpenAIOAuthAccessPolicyChange
}

func (r *openAIOAuthAccessHandlerRepo) ListAccounts(context.Context) ([]service.OpenAIOAuthAccessAccount, error) {
	return r.accounts, nil
}

func (r *openAIOAuthAccessHandlerRepo) ListUsers(context.Context, string, string) ([]service.OpenAIOAuthAccessUser, error) {
	return []service.OpenAIOAuthAccessUser{}, nil
}

func (r *openAIOAuthAccessHandlerRepo) ApplyPolicies(_ context.Context, changes []service.OpenAIOAuthAccessPolicyChange) error {
	r.applied = append([]service.OpenAIOAuthAccessPolicyChange(nil), changes...)
	return nil
}

func TestOpenAIOAuthUserAccessHandlerAppliesRevisionBoundBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &openAIOAuthAccessHandlerRepo{accounts: []service.OpenAIOAuthAccessAccount{{
		ID: 1, Name: "OAuth", Mode: service.OpenAIOAuthUserAccessModePublic,
	}}}
	handler := NewOpenAIOAuthUserAccessHandler(service.NewOpenAIOAuthUserAccessService(repo))
	router := gin.New()
	router.PUT("/policies", handler.Apply)
	body, err := json.Marshal(service.OpenAIOAuthAccessPolicyBatch{Changes: []service.OpenAIOAuthAccessPolicyChange{{
		AccountID: 1, ExpectedRevision: 0, Mode: service.OpenAIOAuthUserAccessModeRestricted,
	}}})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/policies", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, repo.applied, 1)
	require.Equal(t, int64(1), repo.applied[0].AccountID)
}

func TestOpenAIOAuthUserAccessHandlerRejectsMalformedBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewOpenAIOAuthUserAccessHandler(service.NewOpenAIOAuthUserAccessService(&openAIOAuthAccessHandlerRepo{}))
	router := gin.New()
	router.POST("/preview", handler.Preview)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/preview", bytes.NewBufferString("{"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
