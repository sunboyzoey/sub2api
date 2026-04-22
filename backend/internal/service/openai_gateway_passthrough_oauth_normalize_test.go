package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenAIPassthroughOAuthBody_AddsDefaultInstructionsAndStripsUnsupportedFields(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","stream":false,"max_output_tokens":16,"input":[{"role":"user","content":[{"type":"input_text","text":"你好"}]}]}`)

	normalized, changed, err := normalizeOpenAIPassthroughOAuthBody(body, false)
	require.NoError(t, err)
	require.True(t, changed)

	var reqBody map[string]any
	require.NoError(t, json.Unmarshal(normalized, &reqBody))

	require.Equal(t, false, reqBody["store"])
	require.Equal(t, true, reqBody["stream"])
	require.NotContains(t, reqBody, "max_output_tokens")
	require.NotEmpty(t, reqBody["instructions"])
}

func TestNormalizeOpenAIPassthroughOAuthBody_ExtractsDeveloperInstructions(t *testing.T) {
	body := []byte(`{"model":"gpt-5.3-codex","input":[{"role":"developer","content":[{"type":"input_text","text":"You are helpful."}]},{"role":"user","content":[{"type":"input_text","text":"你好"}]}]}`)

	normalized, changed, err := normalizeOpenAIPassthroughOAuthBody(body, false)
	require.NoError(t, err)
	require.True(t, changed)

	var reqBody map[string]any
	require.NoError(t, json.Unmarshal(normalized, &reqBody))
	require.Equal(t, "You are helpful.", reqBody["instructions"])

	input, ok := reqBody["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 1)
}

func TestNormalizeOpenAIPassthroughOAuthBody_CompactDropsStoreAndStream(t *testing.T) {
	body := []byte(`{"model":"gpt-5.3-codex","store":true,"stream":true,"input":[{"role":"user","content":"hi"}]}`)

	normalized, changed, err := normalizeOpenAIPassthroughOAuthBody(body, true)
	require.NoError(t, err)
	require.True(t, changed)

	var reqBody map[string]any
	require.NoError(t, json.Unmarshal(normalized, &reqBody))
	require.NotContains(t, reqBody, "store")
	require.NotContains(t, reqBody, "stream")
	require.NotEmpty(t, reqBody["instructions"])
}
