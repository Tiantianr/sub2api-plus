package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassifyCodexClientProfile_LegacyCompatibilityIsClosedAndExplicit(t *testing.T) {
	legacyOriginators := []string{
		"codex_app",
		"codex_exec",
		"codex_sdk_ts",
		"codex_vscode_copilot",
	}
	for _, originator := range legacyOriginators {
		t.Run(originator, func(t *testing.T) {
			ua := originator + "/0.147.0 (Ubuntu 24.04; x86_64) xterm-256color"
			_, ok := ClassifyCodexClientProfile(ua, originator, false)
			require.False(t, ok, "legacy profile must be disabled by default")

			match, ok := ClassifyCodexClientProfile(ua, originator, true)
			require.True(t, ok)
			require.Equal(t, CodexClientProfileLegacyCompatibility, match.Profile)
			require.Equal(t, originator, match.Originator)
			require.Equal(t, "0.147.0", match.Version)

			_, official := ClassifyOfficialCodexClientProfile(ua, originator)
			require.False(t, official, "legacy profile must never be reported as official")
		})
	}
}

func TestClassifyCodexClientProfile_RejectsLooseLegacyAndOfficialForms(t *testing.T) {
	tests := []struct {
		name       string
		userAgent  string
		originator string
	}{
		{"legacy mixed case", "CODEX_APP/0.147.0", "CODEX_APP"},
		{"legacy header mismatch", "codex_app/0.147.0", "codex_exec"},
		{"legacy originator whitespace", "codex_app/0.147.0", " codex_app "},
		{"legacy incomplete version", "codex_app/0.147", "codex_app"},
		{"legacy arbitrary suffix", "codex_app_evil/0.147.0", "codex_app_evil"},
		{"lowercase product family", "codex Desktop/0.147.0", "codex Desktop"},
		{"official mixed case", "CODEX_CLI_RS/0.147.0", "CODEX_CLI_RS"},
		{"overridden trailer", "cccc/0.147.0 (codex-tui; 0.147.0)", "cccc"},
		{"leading zero version", "codex_cli_rs/01.2.3", "codex_cli_rs"},
		{"leading zero prerelease", "codex_cli_rs/1.2.3-01", "codex_cli_rs"},
		{"empty prerelease identifier", "codex_cli_rs/1.2.3-alpha..1", "codex_cli_rs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ClassifyCodexClientProfile(tt.userAgent, tt.originator, true)
			require.False(t, ok)
		})
	}
}

func TestPairConfiguredCodexClientIdentity_PreservesExactConfiguredUA(t *testing.T) {
	ua := "codex_exec/0.147.0 (Mac OS X 15.0; arm64) iTerm.app"
	_, _, ok := PairConfiguredCodexClientIdentity(ua, false)
	require.False(t, ok)

	match, pairedUA, ok := PairConfiguredCodexClientIdentity(ua, true)
	require.True(t, ok)
	require.Equal(t, CodexClientProfileLegacyCompatibility, match.Profile)
	require.Equal(t, "codex_exec", match.Originator)
	require.Equal(t, ua, pairedUA)
}

func TestClassifyCodexClientProfile_PiAgent(t *testing.T) {
	uaWithVer := "pi/0.85.0 (darwin 24.1.0; arm64)"
	uaOS := "pi (darwin 24.1.0; arm64)"

	// Ingress: Pi is not an official Codex client profile and must not be classified as one.
	_, ok := ClassifyCodexClientProfile(uaWithVer, "pi", false)
	require.False(t, ok, "Pi must not be classified as an inbound Codex profile")

	_, okOS := ClassifyCodexClientProfile(uaOS, "pi", false)
	require.False(t, okOS, "Pi without version must not be classified as an inbound Codex profile")

	_, okBare := ClassifyCodexClientProfile("pi", "pi", false)
	require.False(t, okBare)

	_, official := ClassifyOfficialCodexClientProfile(uaWithVer, "pi")
	require.False(t, official)
	_, officialOS := ClassifyOfficialCodexClientProfile(uaOS, "pi")
	require.False(t, officialOS)

	// Outbound: Pi is recognized and paired as an outbound profile with empty protocol Version.
	matchOS, pairedOS, ok := PairConfiguredCodexClientIdentity(uaOS, false)
	require.True(t, ok)
	require.Equal(t, CodexClientProfilePi, matchOS.Profile)
	require.Equal(t, PiOriginator, matchOS.Originator)
	require.Empty(t, matchOS.Version, "Pi wire identity does not declare a Codex protocol version")
	require.Equal(t, uaOS, pairedOS)

	matchVer, pairedVer, ok := PairConfiguredCodexClientIdentity(uaWithVer, false)
	require.True(t, ok)
	require.Equal(t, CodexClientProfilePi, matchVer.Profile)
	require.Equal(t, PiOriginator, matchVer.Originator)
	require.Empty(t, matchVer.Version, "Configured package version is not a protocol version")
	require.Equal(t, uaWithVer, pairedVer)

	require.True(t, HasCoherentConfiguredClientIdentity(uaOS, "pi"))
	require.True(t, HasCoherentConfiguredClientIdentity(uaWithVer, "pi"))
	require.False(t, HasCoherentConfiguredClientIdentity(uaOS, "codex_cli_rs"))
}
