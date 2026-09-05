//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/openai"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

func TestPiOutboundPath_HTTPForwardHeaders(t *testing.T) {
	piUA := openai.DefaultPiUserAgent // "pi (darwin 24.1.0; arm64)"
	svc := &OpenAIGatewayService{}

	account := &Account{
		ID:       1001,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"user_agent": piUA,
		},
	}

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	body := []byte(`{"model":"gpt-5.4","input":[{"type":"message","content":"hello"}]}`)
	req, err := svc.buildUpstreamRequest(context.Background(), c, account, body, "test-token", true, "", false)
	require.NoError(t, err)

	require.Equal(t, piUA, req.Header.Get("user-agent"))
	require.Equal(t, "pi", req.Header.Get("originator"))
	require.Empty(t, req.Header.Get("version"), "Pi outbound HTTP request must not include Version header")
	require.Empty(t, req.Header.Values("version"), "Version header key must not be present")
	require.Empty(t, req.Header.Values("Version"), "Version header key must not be present")
}

func TestPiOutboundPath_WebSocketCompatibilityAndHeaders(t *testing.T) {
	piUA := openai.DefaultPiUserAgent
	headers := make(http.Header)
	openai.ApplyOutboundClientIdentity(headers, piUA, openai.PiOriginator, "", true)

	require.Equal(t, piUA, headers.Get("User-Agent"))
	require.Equal(t, "pi", headers.Get("Originator"))
	require.Empty(t, headers.Get("Version"), "Version header must not be set for Pi")

	compat := normalizeOpenAIWSHandshakeCompatibility(headers)
	require.Equal(t, piUA, compat.userAgent)
	require.Equal(t, "pi", compat.originator)
	require.Empty(t, compat.version, "WebSocket compatibility key version must be empty for Pi")
}

func TestPiOutboundPath_PATValidationHeaders(t *testing.T) {
	piUA := openai.DefaultPiUserAgent
	identity := openAIOutboundIdentity{
		UserAgent:  piUA,
		Originator: openai.PiOriginator,
		Version:    "",
		Source:     openAIOutboundIdentitySourceAccount,
	}

	req := httptest.NewRequest(http.MethodGet, "https://auth.openai.com/api/accounts/v1/user-auth-credential/whoami", nil)
	openai.ApplyOutboundClientIdentity(req.Header, identity.UserAgent, identity.Originator, identity.Version, true)

	require.Equal(t, piUA, req.Header.Get("user-agent"))
	require.Equal(t, "pi", req.Header.Get("originator"))
	require.Empty(t, req.Header.Get("version"), "PAT validation request must not include Version header for Pi")
	require.Empty(t, req.Header.Values("version"))
}

func TestPiOutboundPath_ModelManifestURL(t *testing.T) {
	piUA := openai.DefaultPiUserAgent
	identity := openAIOutboundIdentity{
		UserAgent:  piUA,
		Originator: openai.PiOriginator,
		Version:    "",
		Source:     openAIOutboundIdentitySourceAccount,
	}

	url, err := buildCodexModelsManifestURL("https://chatgpt.com/backend-api/codex/models", false, identity.Version)
	require.NoError(t, err)
	require.Empty(t, url.Query().Get("client_version"), "Models manifest URL must not contain client_version for Pi")
	require.Empty(t, url.RawQuery, "Query string must be completely empty when clientVersion is empty")
}

func TestPiOutboundPath_PrivacyServiceHeaders(t *testing.T) {
	piUA := openai.DefaultPiUserAgent
	identity := openAIOutboundIdentity{
		UserAgent:  piUA,
		Originator: openai.PiOriginator,
		Version:    "",
		Source:     openAIOutboundIdentitySourceAccount,
	}

	client := req.C()
	r := client.R()
	applyOpenAIPrivacyIdentityHeaders(r, identity)

	require.Equal(t, piUA, r.Headers.Get("User-Agent"))
	require.Equal(t, "pi", r.Headers.Get("Originator"))
	require.Empty(t, r.Headers.Get("Version"), "Privacy request must not set Version header for Pi")
}

func TestPiOutboundPath_EnforceCodexIdentityHeadersWithUA(t *testing.T) {
	headers := make(http.Header)
	headers.Set("user-agent", "codex-tui/0.144.0")
	headers.Set("originator", "codex-tui")
	headers.Set("version", "0.144.0")

	// Apply Pi User-Agent override
	enforceCodexIdentityHeadersWithUA(headers, openai.DefaultPiUserAgent)

	require.Equal(t, openai.DefaultPiUserAgent, headers.Get("user-agent"))
	require.Equal(t, "pi", headers.Get("originator"))
	require.Empty(t, headers.Get("version"), "Version header must be deleted when enforced with Pi UA")
	require.Empty(t, headers.Values("version"))
}
