package routes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuthUserAccessRoutesKeepAdminAndStepUpBoundary(t *testing.T) {
	source, err := os.ReadFile("admin.go")
	require.NoError(t, err)
	text := string(source)
	require.Contains(t, text, `admin.Group("/openai-oauth-access")`)
	require.Contains(t, text, `access.GET("/accounts"`)
	require.Contains(t, text, `access.GET("/users"`)
	require.Contains(t, text, `access.POST("/preview"`)
	require.Contains(t, text, `access.PUT("/policies", gin.HandlerFunc(stepUpAuth)`)
}
