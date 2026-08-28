package handler

import (
	"net/http"
	"strings"
	"sync"

	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const securityAuditCaptureWriterContextKey = "sub2api.security_audit.capture_writer"
const securityAuditCurrentCaptureContextKey = "sub2api.security_audit.current_capture"

type securityAuditCaptureWriter struct {
	gin.ResponseWriter

	mu      sync.Mutex
	capture *securityaudit.ConversationCapture
}

// SecurityAuditResponseCaptureMiddleware installs one no-op writer wrapper for
// all gateway routes. It starts retaining bytes only after Prompt Guard returns
// an applicable conversation capture.
func SecurityAuditResponseCaptureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		writer := &securityAuditCaptureWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Set(securityAuditCaptureWriterContextKey, writer)
		c.Next()
		writer.finish()
	}
}

func (w *securityAuditCaptureWriter) Write(payload []byte) (int, error) {
	written, err := w.ResponseWriter.Write(payload)
	if written > 0 {
		w.current().ObserveHTTP(payload[:written])
	}
	return written, err
}

func (w *securityAuditCaptureWriter) WriteString(payload string) (int, error) {
	written, err := w.ResponseWriter.WriteString(payload)
	if written > 0 {
		w.current().ObserveHTTP([]byte(payload[:written]))
	}
	return written, err
}

func (w *securityAuditCaptureWriter) activate(capture *securityaudit.ConversationCapture) {
	if w == nil || capture == nil {
		return
	}
	w.mu.Lock()
	prior := w.capture
	w.capture = capture
	w.mu.Unlock()
	if prior != nil && prior != capture {
		prior.Abort()
	}
}

func (w *securityAuditCaptureWriter) current() *securityaudit.ConversationCapture {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.capture
}

func (w *securityAuditCaptureWriter) finish() {
	if w == nil {
		return
	}
	capture := w.current()
	if capture == nil {
		return
	}
	status := w.Status()
	if status == 0 {
		status = http.StatusOK
	}
	capture.FinishHTTP(status, w.Header().Get("Content-Type"))
}

func attachSecurityAuditConversationCapture(c *gin.Context, decision *securityaudit.Decision) {
	if c == nil || decision == nil || !decision.AllowNextStage || decision.Prompt == nil || decision.Prompt.Capture == nil {
		return
	}
	capture := decision.Prompt.Capture
	c.Set(securityAuditCurrentCaptureContextKey, capture)
	if value, exists := c.Get(securityAuditCaptureWriterContextKey); exists {
		if writer, ok := value.(*securityAuditCaptureWriter); ok {
			writer.activate(capture)
		}
	}
}

func currentSecurityAuditConversationCapture(c *gin.Context) *securityaudit.ConversationCapture {
	if c == nil {
		return nil
	}
	value, exists := c.Get(securityAuditCurrentCaptureContextKey)
	if !exists {
		return nil
	}
	capture, _ := value.(*securityaudit.ConversationCapture)
	return capture
}

func observeSecurityAuditConversationFrame(c *gin.Context, payload []byte) {
	capture := currentSecurityAuditConversationCapture(c)
	if capture == nil {
		return
	}
	capture.ObserveFrame(payload)
	eventType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "type").String()))
	switch eventType {
	case "response.completed", "response.done":
		responseID := strings.TrimSpace(gjson.GetBytes(payload, "response.id").String())
		if responseID == "" {
			responseID = strings.TrimSpace(gjson.GetBytes(payload, "id").String())
		}
		capture.FinishTurn(responseID)
	case "error", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		capture.Abort()
	}
}

func completeSecurityAuditConversationWithoutOutput(c *gin.Context, parentAlias string) {
	if capture := currentSecurityAuditConversationCapture(c); capture != nil {
		capture.CompleteWithoutOutput(parentAlias)
	}
}

func abortSecurityAuditConversationCapture(c *gin.Context) {
	if capture := currentSecurityAuditConversationCapture(c); capture != nil {
		capture.Abort()
	}
}
