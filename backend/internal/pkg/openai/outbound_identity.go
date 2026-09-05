package openai

import (
	"net/http"
	"strings"
)

// ApplyOutboundClientIdentity serializes an already resolved administrator
// identity. Pi never declares a Codex protocol version. Public API requests omit
// Originator and retain Version only if an endpoint explicitly staged one.
func ApplyOutboundClientIdentity(headers http.Header, userAgent, originator, version string, useCodexIdentity bool) {
	if headers == nil {
		return
	}
	hadVersion := false
	for name, values := range headers {
		switch {
		case strings.EqualFold(name, "Version"):
			for _, value := range values {
				hadVersion = hadVersion || strings.TrimSpace(value) != ""
			}
			delete(headers, name)
		case strings.EqualFold(name, "User-Agent"), strings.EqualFold(name, "Originator"):
			delete(headers, name)
		}
	}
	headers.Set("User-Agent", userAgent)
	if useCodexIdentity {
		headers.Set("Originator", originator)
	}
	if originator != PiOriginator && version != "" && (useCodexIdentity || hadVersion) {
		headers.Set("Version", version)
	}
}
