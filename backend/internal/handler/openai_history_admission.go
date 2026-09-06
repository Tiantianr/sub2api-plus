package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

func (h *OpenAIGatewayHandler) prepareOpenAIHistory(c *gin.Context, apiKey *service.APIKey, protocol string, body []byte, sessionHash string, anthropic bool, readOnly ...bool) bool {
	if openAICompatibleRequestPlatform(c.Request.Context(), apiKey) != service.PlatformOpenAI {
		return true
	}
	if err := h.gatewayService.PrepareOpenAIHistoryRequest(c, protocol, body, sessionHash, readOnly...); err != nil {
		if !h.handleOpenAIHistoryError(c, err, anthropic, false) {
			if anthropic {
				h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to classify conversation history")
			} else {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to classify conversation history")
			}
		}
		return false
	}
	return true
}

func openAIHistoryErrorDetails(err error) (int, string, string, string, bool) {
	switch {
	case errors.Is(err, service.ErrOpenAIExternalHistory):
		return http.StatusBadRequest, "invalid_request_error", "external_history_not_allowed", service.ErrOpenAIExternalHistory.Error(), true
	case errors.Is(err, service.ErrOpenAIHistoryUnavailable):
		return http.StatusServiceUnavailable, "api_error", "conversation_ownership_unavailable", service.ErrOpenAIHistoryUnavailable.Error(), true
	case errors.Is(err, service.ErrOpenAIHistoryConflict):
		return http.StatusConflict, "invalid_request_error", "conversation_routing_conflict", service.ErrOpenAIHistoryConflict.Error(), true
	default:
		return 0, "", "", "", false
	}
}

func (h *OpenAIGatewayHandler) handleOpenAIHistoryError(c *gin.Context, err error, anthropic, streamStarted bool) bool {
	status, errType, code, message, ok := openAIHistoryErrorDetails(err)
	if !ok {
		return false
	}
	if errors.Is(err, service.ErrOpenAIExternalHistory) {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
	}
	if anthropic {
		h.anthropicStreamingAwareError(c, status, errType, message, streamStarted)
	} else {
		h.handleStreamingAwareErrorWithCode(c, status, errType, code, message, streamStarted, false)
	}
	return true
}

func writeOpenAIHistoryWSError(ctx context.Context, conn *coderws.Conn, err error, reconnect bool) bool {
	_, errType, code, message, ok := openAIHistoryErrorDetails(err)
	if !ok {
		return false
	}
	if reconnect && errors.Is(err, service.ErrOpenAIExternalHistory) {
		code = "conversation_reconnect_required"
		message = "This account cannot continue the supplied history; reconnect to select another account."
	}
	payload, marshalErr := json.Marshal(gin.H{"type": "error", "error": gin.H{
		"type": errType, "code": code, "message": message,
	}})
	if marshalErr == nil {
		writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = conn.Write(writeCtx, coderws.MessageText, payload)
	}
	return true
}
