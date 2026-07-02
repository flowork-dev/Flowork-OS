// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package router

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flowork-os/flowork_Router/internal/store"
)

func streamAnthropicWithTools(ctx context.Context, p *store.ProviderConnection, baseURL string, req OpenAIRequest, w http.ResponseWriter, flusher http.Flusher) (OpenAIUsage, int, error) {
	req.Stream = true
	body, err := buildAnthropicToolBody(req)
	if err != nil {
		return OpenAIUsage{}, 0, fmt.Errorf("build anthropic tool body: %w", err)
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return OpenAIUsage{}, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("User-Agent", "claude-cli/1.0.0 (flow_router)")
	if err := applyAuth(httpReq, p); err != nil {
		return OpenAIUsage{}, http.StatusUnauthorized, err
	}
	resp, err := outboundClient(ctx).Do(httpReq)
	if err != nil {
		return OpenAIUsage{}, http.StatusBadGateway, fmt.Errorf("upstream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return OpenAIUsage{}, resp.StatusCode, fmt.Errorf("anthropic %d: %s", resp.StatusCode, truncate(string(b), 200))
	}

	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()
	writeOpenAIDelta(w, flusher, chunkID, created, req.Model, map[string]any{"role": "assistant"}, "")

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 4*1024*1024)
	var usage OpenAIUsage
	var firstWritten bool
	stopReason := ""
	blockToTool := map[int]int{}
	toolIdx := -1
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if payload == "" {
			continue
		}
		var ev struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
			Message struct {
				Usage struct {
					InputTokens              int `json:"input_tokens"`
					CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
					CacheReadInputTokens     int `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal([]byte(payload), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "message_start":
			usage.PromptTokens = ev.Message.Usage.InputTokens + ev.Message.Usage.CacheReadInputTokens + ev.Message.Usage.CacheCreationInputTokens
			if ev.Message.Usage.CacheReadInputTokens > 0 || ev.Message.Usage.CacheCreationInputTokens > 0 {
				log.Printf("flow_router anthropic cache (stream): read=%d create=%d fresh_input=%d",
					ev.Message.Usage.CacheReadInputTokens, ev.Message.Usage.CacheCreationInputTokens, ev.Message.Usage.InputTokens)
			}
		case "content_block_start":
			if ev.ContentBlock.Type == "tool_use" {
				toolIdx++
				blockToTool[ev.Index] = toolIdx
				writeOpenAIDelta(w, flusher, chunkID, created, req.Model, map[string]any{
					"tool_calls": []map[string]any{{
						"index": toolIdx, "id": ev.ContentBlock.ID, "type": "function",
						"function": map[string]any{"name": ev.ContentBlock.Name, "arguments": ""},
					}},
				}, "")
				firstWritten = true
			}
		case "content_block_delta":
			switch ev.Delta.Type {
			case "text_delta":
				if ev.Delta.Text != "" {
					writeOpenAIDelta(w, flusher, chunkID, created, req.Model, map[string]any{"content": ev.Delta.Text}, "")
					firstWritten = true
				}
			case "input_json_delta":
				if ev.Delta.PartialJSON != "" {
					writeOpenAIDelta(w, flusher, chunkID, created, req.Model, map[string]any{
						"tool_calls": []map[string]any{{
							"index":    blockToTool[ev.Index],
							"function": map[string]any{"arguments": ev.Delta.PartialJSON},
						}},
					}, "")
					firstWritten = true
				}
			}
		case "message_delta":
			if ev.Delta.StopReason != "" {
				stopReason = ev.Delta.StopReason
			}
			if ev.Usage.OutputTokens > 0 {
				usage.CompletionTokens = ev.Usage.OutputTokens
			}
		case "message_stop":
			fr := "stop"
			switch stopReason {
			case "max_tokens":
				fr = "length"
			case "tool_use":
				fr = "tool_calls"
			}
			writeOpenAIDelta(w, flusher, chunkID, created, req.Model, map[string]any{}, fr)
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
			return usage, http.StatusOK, nil
		}
	}
	if err := scanner.Err(); err != nil && !firstWritten {
		return usage, http.StatusBadGateway, fmt.Errorf("anthropic stream read: %w", err)
	}
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage, http.StatusOK, nil
}

func forwardAnthropicWithTools(ctx context.Context, p *store.ProviderConnection, baseURL string, req OpenAIRequest) (*OpenAIResponse, int, error) {
	req.Stream = false
	body, err := buildAnthropicToolBody(req)
	if err != nil {
		return nil, 0, fmt.Errorf("build anthropic tool body: %w", err)
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build req: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("User-Agent", "claude-cli/1.0.0 (flow_router)")
	if err := applyAuth(httpReq, p); err != nil {
		return nil, http.StatusUnauthorized, err
	}
	resp, err := outboundClient(ctx).Do(httpReq)
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("upstream: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("anthropic %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	out, err := parseAnthropicToolResponse(respBody, req.Model)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	return out, http.StatusOK, nil
}

func hasToolContext(req OpenAIRequest) bool {
	if len(req.Tools) > 0 && string(req.Tools) != "null" {
		return true
	}
	for _, m := range req.Messages {
		if len(m.ToolCalls) > 0 || m.ToolCallID != "" || m.Role == "tool" {
			return true
		}
	}
	return false
}

type openAIToolFn struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// anthropicUserContentHook — ⭐ SEAM (Rule 7 POLA B, owner-approved 2026-07-02): bentuk
// content USER string → block-array Anthropic (mis. vision: text + image base64) TANPA
// buka file frozen ini. Default nil → string apa adanya (perilaku lama). Diisi sibling
// non-frozen (vision_anthropic_ext.go); sibling dihapus → balik default aman.
// 📄 Dok: FLowork_os/lock/chat-vision.md
var anthropicUserContentHook func(content string) any

func buildAnthropicToolBody(req OpenAIRequest) ([]byte, error) {
	body := map[string]any{
		"model":      normalizeClaudeModel(req.Model),
		"max_tokens": req.MaxTokens,
	}
	if body["max_tokens"].(int) <= 0 {
		body["max_tokens"] = 4096
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		body["top_p"] = req.TopP
	}
	if req.Stream {
		body["stream"] = true
	}

	var sysParts []string
	var messages []map[string]any
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			sysParts = append(sysParts, m.Content)
		case "tool":

			messages = append(messages, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.Content,
				}},
			})
		case "assistant":
			if len(m.ToolCalls) > 0 {
				var calls []openAIToolCall
				if err := json.Unmarshal(m.ToolCalls, &calls); err == nil && len(calls) > 0 {
					blocks := []map[string]any{}
					if strings.TrimSpace(m.Content) != "" {
						blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
					}
					for _, c := range calls {
						var input any
						if c.Function.Arguments != "" {
							_ = json.Unmarshal([]byte(c.Function.Arguments), &input)
						}
						if input == nil {
							input = map[string]any{}
						}
						blocks = append(blocks, map[string]any{
							"type":  "tool_use",
							"id":    c.ID,
							"name":  c.Function.Name,
							"input": input,
						})
					}
					messages = append(messages, map[string]any{"role": "assistant", "content": blocks})
					continue
				}
			}
			messages = append(messages, map[string]any{"role": "assistant", "content": m.Content})
		case "user":
			content := any(m.Content)
			if anthropicUserContentHook != nil {
				if v := anthropicUserContentHook(m.Content); v != nil {
					content = v
				}
			}
			messages = append(messages, map[string]any{"role": "user", "content": content})
		}
	}
	cacheOn := promptCacheEnabled()
	if len(sysParts) > 0 {
		if cacheOn {
			// DYNAMIC BOUNDARY: system block PERTAMA = persona STABIL (Tier1+2, mr-flow
			// pecah di marker) → cache_control cuma di situ. Sisanya (volatile Tier3/recall
			// + enrichment) fresh. Enrichment masuk SETELAH stabil (injectSystem cache-on)
			// → prefix stabil ga batal → cache_read gede lintas-turn.
			blocks := make([]map[string]any, 0, len(sysParts))
			for i, p := range sysParts {
				blk := map[string]any{"type": "text", "text": p}
				if i == 0 {
					blk["cache_control"] = map[string]any{"type": "ephemeral"}
				}
				blocks = append(blocks, blk)
			}
			body["system"] = blocks
		} else {
			body["system"] = strings.Join(sysParts, "\n\n")
		}
	}
	msgs := mergeConsecutiveAnthropic(messages)
	if cacheOn {
		markLastBlockCache(msgs) // cache prefix percakapan (history) inkremental tiap turn
	}
	body["messages"] = msgs

	if len(req.Tools) > 0 && string(req.Tools) != "null" {
		var oaTools []openAIToolFn
		if err := json.Unmarshal(req.Tools, &oaTools); err == nil {
			var anthTools []map[string]any
			for _, t := range oaTools {
				if t.Function.Name == "" {
					continue
				}
				schema := t.Function.Parameters
				if len(schema) == 0 || string(schema) == "null" {
					schema = json.RawMessage(`{"type":"object","properties":{}}`)
				}
				anthTools = append(anthTools, map[string]any{
					"name":         t.Function.Name,
					"description":  t.Function.Description,
					"input_schema": schema,
				})
			}
			if len(anthTools) > 0 {
				if cacheOn {
					// Breakpoint di tool TERAKHIR → cache seluruh array tool-schema
					// (stabil lintas-turn). Anthropic cache prefix s/d breakpoint ini.
					anthTools[len(anthTools)-1]["cache_control"] = map[string]any{"type": "ephemeral"}
				}
				body["tools"] = anthTools
			}
		}

		if tc := convertToolChoice(req.ToolChoice); tc != nil {
			body["tool_choice"] = tc
		}
	}
	return json.Marshal(body)
}

// mergeConsecutiveAnthropic menggabung pesan berurutan dengan role sama menjadi
// SATU pesan berisi array content-block. AKAR fix: Anthropic mewajibkan role
// user/assistant selang-seling; beberapa tool_result beruntun (parallel tool calls)
// atau teks-user setelah tool_result = 2 pesan same-role beruntun → HTTP 400
// "messages: roles must alternate". Dulu disiasati paksa 1 tool/turn (sequential,
// lambat, gampang timeout). Sekarang hasil parallel dikemas bener → banyak
// tool_result boleh dalam 1 user message, model bisa minta banyak tool sekaligus.
func mergeConsecutiveAnthropic(in []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, m := range in {
		if n := len(out); n > 0 && out[n-1]["role"] == m["role"] {
			out[n-1]["content"] = append(anthropicBlocks(out[n-1]["content"]), anthropicBlocks(m["content"])...)
			continue
		}
		out = append(out, m)
	}
	return out
}

// anthropicBlocks menormalkan content (string ATAU []map[string]any) jadi block-array
// biar bisa digabung lintas-pesan tanpa kehilangan tipe (text/tool_result/tool_use).
func anthropicBlocks(content any) []map[string]any {
	switch c := content.(type) {
	case []map[string]any:
		return c
	case string:
		if strings.TrimSpace(c) == "" {
			return nil
		}
		return []map[string]any{{"type": "text", "text": c}}
	}
	return nil
}

// promptCacheEnabled — switch GUI FLOWORK_PROMPT_CACHE (default ON). Prompt caching
// Anthropic udah GA: cukup `cache_control` di body (anthropic-version 2023-06-01),
// TANPA header beta / tanpa nyentuh peniruan auth langganan. Set "0"/"false"/"off"
// buat matiin kalau ada provider yg nolak cache_control.
func promptCacheEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FLOWORK_PROMPT_CACHE")))
	return v != "0" && v != "false" && v != "off"
}

// markLastBlockCache naruh cache_control di block TERAKHIR pesan terakhir → Anthropic
// cache prefix percakapan sepanjang mungkin (history statis di-reuse turn berikut).
func markLastBlockCache(msgs []map[string]any) {
	if len(msgs) == 0 {
		return
	}
	last := msgs[len(msgs)-1]
	blocks := anthropicBlocks(last["content"])
	if len(blocks) == 0 {
		return
	}
	blocks[len(blocks)-1]["cache_control"] = map[string]any{"type": "ephemeral"}
	last["content"] = blocks
}

func convertToolChoice(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		switch s {
		case "auto":
			return map[string]any{"type": "auto"}
		case "required":
			return map[string]any{"type": "any"}
		case "none":
			return nil
		}
	}
	var obj struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Function.Name != "" {
		return map[string]any{"type": "tool", "name": obj.Function.Name}
	}
	return nil
}

type anthropicRichResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

func parseAnthropicToolResponse(respBody []byte, reqModel string) (*OpenAIResponse, error) {
	var ar anthropicRichResponse
	if err := json.Unmarshal(respBody, &ar); err != nil {
		return nil, fmt.Errorf("parse anthropic rich: %w", err)
	}
	var textParts []string
	var toolCalls []map[string]any
	for _, c := range ar.Content {
		switch c.Type {
		case "text":
			textParts = append(textParts, c.Text)
		case "tool_use":
			args := "{}"
			if len(c.Input) > 0 {
				args = string(c.Input)
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   c.ID,
				"type": "function",
				"function": map[string]any{
					"name":      c.Name,
					"arguments": args,
				},
			})
		}
	}
	finish := "stop"
	switch ar.StopReason {
	case "max_tokens":
		finish = "length"
	case "tool_use":
		finish = "tool_calls"
	}
	msg := map[string]any{
		"role":    "assistant",
		"content": strings.Join(textParts, ""),
	}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
		if msg["content"] == "" {
			msg["content"] = nil
		}
	}

	out := map[string]any{
		"id":      ar.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   reqModel,
		"choices": []map[string]any{{
			"index":         0,
			"message":       msg,
			"finish_reason": finish,
		}},
		"usage": map[string]any{
			"prompt_tokens":     ar.Usage.InputTokens + ar.Usage.CacheReadInputTokens + ar.Usage.CacheCreationInputTokens,
			"completion_tokens": ar.Usage.OutputTokens,
			"total_tokens":      ar.Usage.InputTokens + ar.Usage.CacheReadInputTokens + ar.Usage.CacheCreationInputTokens + ar.Usage.OutputTokens,
			"prompt_tokens_details": map[string]any{
				"cached_tokens":         ar.Usage.CacheReadInputTokens,
				"cache_creation_tokens": ar.Usage.CacheCreationInputTokens,
			},
		},
	}
	// Observability: bukti caching kepakai (cache_read>0 = hemat token turn ini).
	if ar.Usage.CacheReadInputTokens > 0 || ar.Usage.CacheCreationInputTokens > 0 {
		log.Printf("flow_router anthropic cache: read=%d create=%d fresh_input=%d",
			ar.Usage.CacheReadInputTokens, ar.Usage.CacheCreationInputTokens, ar.Usage.InputTokens)
	}
	raw, _ := json.Marshal(out)
	var resp OpenAIResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
