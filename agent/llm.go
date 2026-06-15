// llm.go — helper shared buat panggil router (LLM) dengan tool_choice DIPAKSA.
// Dipakai CODER (design spec) + VERIFIER (LLM-judge). Pola classifier mr-flow:
// forced-tool → output terstruktur, anti free-text halu. Loopback ke router :2402.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// routerForcedTool — POST ke router, PAKSA LLM manggil `toolName` (1x) → balik
// arguments JSON mentah. Caller unmarshal sendiri ke struct-nya.
func routerForcedTool(ctx context.Context, model, systemPrompt, userPrompt string, tool map[string]any, toolName string, maxTokens int) (json.RawMessage, error) {
	reqMap := map[string]any{
		"model": model,
		"messages": []any{
			map[string]any{"role": "system", "content": systemPrompt},
			map[string]any{"role": "user", "content": userPrompt},
		},
		"tools":       []any{tool},
		"tool_choice": map[string]any{"type": "function", "function": map[string]any{"name": toolName}},
		"max_tokens":  maxTokens,
	}
	body, _ := json.Marshal(reqMap)
	hreq, _ := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:2402/v1/chat/completions", bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 180 * time.Second}).Do(hreq)
	if err != nil {
		return nil, fmt.Errorf("router call: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("router status %d: %s", resp.StatusCode, trimStr(string(raw), 200))
	}
	var oResp struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					Function struct {
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &oResp); err != nil {
		return nil, fmt.Errorf("decode router resp: %w", err)
	}
	if len(oResp.Choices) == 0 || len(oResp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("LLM ga manggil %s (forced-tool gagal)", toolName)
	}
	return json.RawMessage(oResp.Choices[0].Message.ToolCalls[0].Function.Arguments), nil
}

// chatToolCall — one tool the LLM decided to call during a chat turn.
type chatToolCall struct {
	ID        string
	Name      string
	Arguments string // raw JSON string
}

// chatResult — a chat turn's reply: free-text content and/or tool calls.
type chatResult struct {
	Content   string
	ToolCalls []chatToolCall
	Raw       json.RawMessage // the raw assistant message object, for threading back
}

// routerChat — a GENERAL multi-turn chat call (this is "how Claude works": the FULL
// messages[] is sent every turn, optional tools with tool_choice=auto, big model holds
// the whole context). Returns the assistant's text and/or tool calls. messages is the
// OpenAI-shape array the caller threads (system + full history + tool results). Used by
// the server-side Architect chat brain. Long timeout (deep reasoning + rate-limit
// failover); honors ctx.
func routerChat(ctx context.Context, model string, messages []map[string]any, tools []map[string]any, maxTokens int) (chatResult, error) {
	var res chatResult
	reqMap := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	if len(tools) > 0 {
		reqMap["tools"] = tools
		reqMap["tool_choice"] = "auto"
	}
	body, _ := json.Marshal(reqMap)
	hreq, _ := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:2402/v1/chat/completions", bytes.NewReader(body))
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 280 * time.Second}).Do(hreq)
	if err != nil {
		return res, fmt.Errorf("router chat: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return res, fmt.Errorf("router status %d: %s", resp.StatusCode, trimStr(string(raw), 240))
	}
	var oResp struct {
		Choices []struct {
			Message struct {
				Content   string          `json:"content"`
				ToolCalls json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &oResp); err != nil {
		return res, fmt.Errorf("decode chat resp: %w", err)
	}
	if len(oResp.Choices) == 0 {
		return res, fmt.Errorf("router returned no choices")
	}
	msg := oResp.Choices[0].Message
	res.Content = msg.Content
	res.Raw = msg.ToolCalls
	if len(msg.ToolCalls) > 0 && string(msg.ToolCalls) != "null" {
		var tcs []struct {
			ID       string `json:"id"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		}
		if e := json.Unmarshal(msg.ToolCalls, &tcs); e == nil {
			for _, tc := range tcs {
				res.ToolCalls = append(res.ToolCalls, chatToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: tc.Function.Arguments})
			}
		}
	}
	return res, nil
}
