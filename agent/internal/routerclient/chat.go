// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval (autonomy grant 2026-06-19).
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-20
// Reason: CGM router chat-completion helper (LLMFunc source) — built + used live in
//   P1 prove-loop + re-distil batch (685 instinct). Extend = new file, jangan modify ini.
//
// chat.go — agent → router chat-completion helper (CGM digestion extractor source).
//
// File BARU (routerclient.go locked). Reuse Client (BaseURL whitelisted, HTTP).
// Manggil router POST /v1/chat/completions (OpenAI-compatible) buat dapet teks
// completion dari prompt. Dipakai sebagai LLMFunc closure di CGM digestion
// (agentdb.DigestDeps.LLM) — extractor 5W1H butuh completion, bukan embedding.
//
// Layering: agentdb gak import routerclient (func-type injection). Caller di
// lapis non-beku (agentmgr) yang nyolokin closure ini → DigestPendingInteractions.
//
// Kenapa di sini (bukan agentmgr): biar 1 tempat konsisten ngomong ke router
// (sama kayak EmbedText/SearchBrain), URL whitelist + timeout sentral.

package routerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultChatModel — model reasoning default (flowork-brain = LLM lokal di router).
const DefaultChatModel = "flowork-brain"

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatReq struct {
	Model       string    `json:"model"`
	Messages    []chatMsg `json:"messages"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature"`
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ChatComplete minta 1 completion dari router buat `prompt`. model kosong →
// DefaultChatModel. maxTokens<=0 → biar router/model yang nentuin. Timeout
// di-extend (extraction prompt bisa lama di model lokal) lewat ctx caller —
// kalau ctx gak punya deadline, fallback ke timeout panjang sendiri.
//
// Return teks mentah (caller yang parse JSON terkekang via agentdb.ParseExtraction).
func (c *Client) ChatComplete(ctx context.Context, model, prompt string, maxTokens int) (string, error) {
	if c == nil {
		return "", fmt.Errorf("router client nil")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt kosong")
	}
	if model == "" {
		model = DefaultChatModel
	}

	body, err := json.Marshal(chatReq{
		Model:       model,
		Messages:    []chatMsg{{Role: "user", Content: prompt}},
		Stream:      false,
		MaxTokens:   maxTokens,
		Temperature: 0, // deterministik buat ekstraksi (anti-halu, JSON terkekang)
	})
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	// Model lokal bisa lambat buat extraction → kasih deadline panjang kalau ctx
	// belum punya. Jangan ngandelin DefaultTimeout (30s) yang dipake embedding.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// HTTP.Do timeout (30s) bisa lebih ketat dari ctx; pakai client tanpa timeout
	// global supaya ctx yang nentuin (extraction butuh > 30s di model lokal).
	hc := &http.Client{Timeout: 0}
	resp, err := hc.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("chat: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB cap
	var out chatResp
	if uerr := json.Unmarshal(raw, &out); uerr != nil {
		return "", fmt.Errorf("decode (status %d): %w", resp.StatusCode, uerr)
	}
	if resp.StatusCode >= 400 {
		if out.Error != nil {
			return "", fmt.Errorf("router %d: %s", resp.StatusCode, out.Error.Message)
		}
		return "", fmt.Errorf("router status %d", resp.StatusCode)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("empty completion (model %q configured?)", model)
	}
	return out.Choices[0].Message.Content, nil
}
