package routes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentModerationInputDetailRouteStaysInsideAdminBoundary(t *testing.T) {
	source, err := os.ReadFile("admin.go")
	require.NoError(t, err)
	text := string(source)
	require.Contains(t, text, `risk := admin.Group("/risk-control")`)
	require.Contains(t, text, `risk.GET("/logs/:id/input", h.Admin.ContentModeration.GetLogInput)`)
}
