package anthropic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func newHeaderCaptureServer(t *testing.T) (*httptest.Server, <-chan http.Header) {
	t.Helper()
	captured := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockAnthropicGenerateResponse())
	}))
	t.Cleanup(server.Close)
	return server, captured
}

func generateOnce(t *testing.T, opts ...Option) {
	t.Helper()
	p, err := New(opts...)
	require.NoError(t, err)
	model, err := p.LanguageModel(t.Context(), "claude-sonnet-4-20250514")
	require.NoError(t, err)
	_, err = model.Generate(t.Context(), fantasy.Call{Prompt: fantasy.Prompt{{
		Role:    fantasy.MessageRoleUser,
		Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Hi"}},
	}}})
	require.NoError(t, err)
}

func clearAnthropicEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_PROFILE",
		"ANTHROPIC_CUSTOM_HEADERS",
	} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	t.Setenv("ANTHROPIC_CONFIG_DIR", t.TempDir())
}

func TestProviderIgnoresAnthropicSDKProfile(t *testing.T) {
	clearAnthropicEnvironment(t)
	configDir := os.Getenv("ANTHROPIC_CONFIG_DIR")
	profile := `{
		"authentication": {"type": "user_oauth"},
		"base_url": "http://127.0.0.1:1",
		"workspace_id": "wrkspc_leaked"
	}`
	credentials := `{"type": "oauth_token", "access_token": "profile-token"}`
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "configs"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "credentials"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "configs", "default.json"), []byte(profile), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "credentials", "default.json"), []byte(credentials), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "active_config"), []byte("default"), 0o600))

	server, captured := newHeaderCaptureServer(t)
	generateOnce(t, WithAPIKey("configured-key"), WithBaseURL(server.URL))

	headers := <-captured
	require.Equal(t, "configured-key", headers.Get("X-Api-Key"))
	require.Empty(t, headers.Get("Anthropic-Workspace-Id"))
	require.Empty(t, headers.Get("Authorization"))
}

func TestProviderIgnoresAnthropicEnvironmentCredentials(t *testing.T) {
	clearAnthropicEnvironment(t)
	t.Setenv("ANTHROPIC_API_KEY", "env-key")
	t.Setenv("ANTHROPIC_BASE_URL", "http://127.0.0.1:1")

	server, captured := newHeaderCaptureServer(t)
	generateOnce(t,
		WithBaseURL(server.URL),
		WithHeaders(map[string]string{"Authorization": "Bearer configured-token"}),
	)

	headers := <-captured
	require.Empty(t, headers.Get("X-Api-Key"))
	require.Equal(t, "Bearer configured-token", headers.Get("Authorization"))
}
