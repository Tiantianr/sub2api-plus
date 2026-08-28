package admin

import (
	"net/http"
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

const openAIOAuthAccessRequestBodyLimit = 2 << 20

type OpenAIOAuthUserAccessHandler struct {
	service *service.OpenAIOAuthUserAccessService
}

func NewOpenAIOAuthUserAccessHandler(service *service.OpenAIOAuthUserAccessService) *OpenAIOAuthUserAccessHandler {
	return &OpenAIOAuthUserAccessHandler{service: service}
}

func (h *OpenAIOAuthUserAccessHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.service.ListAccounts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, accounts)
}

func (h *OpenAIOAuthUserAccessHandler) ListUsers(c *gin.Context) {
	page := queryPositiveInt(c, "page", 1)
	limit := queryPositiveInt(c, "limit", 50)
	result, err := h.service.ListUsers(c.Request.Context(), service.OpenAIOAuthAccessUserFilter{
		Search: c.Query("search"),
		Status: c.Query("status"),
		Access: c.Query("access"),
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OpenAIOAuthUserAccessHandler) Preview(c *gin.Context) {
	batch, ok := bindOpenAIOAuthAccessBatch(c)
	if !ok {
		return
	}
	preview, err := h.service.Preview(c.Request.Context(), batch)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{
		"account_count":       len(preview.Accounts),
		"grant_added_count":   preview.GrantAddedCount,
		"grant_removed_count": preview.GrantRemovedCount,
	})
	response.Success(c, preview)
}

func (h *OpenAIOAuthUserAccessHandler) Apply(c *gin.Context) {
	batch, ok := bindOpenAIOAuthAccessBatch(c)
	if !ok {
		return
	}
	result, err := h.service.Apply(c.Request.Context(), batch)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditExtra(c, map[string]any{
		"account_count":       result.AccountCount,
		"grant_added_count":   result.GrantAddedCount,
		"grant_removed_count": result.GrantRemovedCount,
		"account_ids":         result.AuditAccountIDs,
		"revision_changes":    result.AuditRevisions,
		"mode_changes":        result.AuditModes,
	})
	response.Success(c, result)
}

func bindOpenAIOAuthAccessBatch(c *gin.Context) (service.OpenAIOAuthAccessPolicyBatch, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, openAIOAuthAccessRequestBodyLimit)
	var batch service.OpenAIOAuthAccessPolicyBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		response.BadRequest(c, "invalid OpenAI OAuth access policy request")
		return service.OpenAIOAuthAccessPolicyBatch{}, false
	}
	return batch, true
}

func queryPositiveInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
