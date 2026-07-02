// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package router

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flowork-os/flowork_Router/internal/creds"
	"github.com/flowork-os/flowork_Router/internal/executors"
	"github.com/flowork-os/flowork_Router/internal/providercompat"
	"github.com/flowork-os/flowork_Router/internal/safego"
	"github.com/flowork-os/flowork_Router/internal/store"
)

// httpTimeout — timeout request upstream LLM (dispatcher/proxy). SWITCH FLOWORK_ROUTER_HTTP_TIMEOUT
// (detik), default 300s — di-baca saat boot (http.Client di-set sekali; ganti = restart).
var httpTimeout = func() time.Duration {
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("FLOWORK_ROUTER_HTTP_TIMEOUT"))); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return 300 * time.Second
}()

var httpClient = &http.Client{Timeout: httpTimeout}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`

	Tools      json.RawMessage `json:"tools,omitempty"`
	ToolChoice json.RawMessage `json:"tool_choice,omitempty"`

	TopK              int             `json:"top_k,omitempty"`
	MaxCompletionTok  int             `json:"max_completion_tokens,omitempty"`
	Thinking          json.RawMessage `json:"thinking,omitempty"`
	Reasoning         json.RawMessage `json:"reasoning,omitempty"`
	EnableThinking    *bool           `json:"enable_thinking,omitempty"`
	PresencePenalty   float64         `json:"presence_penalty,omitempty"`
	FrequencyPenalty  float64         `json:"frequency_penalty,omitempty"`
	Seed              *int64          `json:"seed,omitempty"`
	Stop              json.RawMessage `json:"stop,omitempty"`
	ResponseFormat    json.RawMessage `json:"response_format,omitempty"`
	Prediction        json.RawMessage `json:"prediction,omitempty"`
	Store             *bool           `json:"store,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	N                 int             `json:"n,omitempty"`
	Logprobs          *bool           `json:"logprobs,omitempty"`
	TopLogprobs       *int            `json:"top_logprobs,omitempty"`
	LogitBias         json.RawMessage `json:"logit_bias,omitempty"`
	User              string          `json:"user,omitempty"`
	ParallelToolCalls *bool           `json:"parallel_tool_calls,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`

	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	PromptTokensDetails *OpenAIPromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

type OpenAIPromptTokensDetails struct {
	CachedTokens        int `json:"cached_tokens,omitempty"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// applyInjectShaper — ⭐ SEAM (Rule 7 POLA B, 2026-07-02 owner-approved): pembentuk
// request PASCA semua injeksi & filter tool (budget agregat system-inject, sticky-union
// tool cache-aman, reorder cache-aware masa depan). Default no-op = perilaku lama.
// Override via sibling non-frozen (inject_budget_ext.go / tools_sticky_ext.go) TANPA
// buka file frozen ini. Sibling dihapus → balik no-op aman.
// 📄 Dok: FLowork_os/lock/prompt-diet.md
var applyInjectShaper = func(ctx context.Context, req OpenAIRequest, settings *store.Settings) OpenAIRequest {
	return req
}

func DispatchChatCompletion(ctx context.Context, req OpenAIRequest) (*OpenAIResponse, int, error) {
	d, err := store.Open()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("store open: %w", err)
	}

	settings, _ := store.LoadSettings(d)

	if req.Model == "" {
		if top := globalFallbackModels(d, nil); len(top) > 0 {
			req.Model = top[0]
		} else if settings != nil {
			req.Model = settings.DefaultModel
		}
	}

	if settings != nil && settings.RtkTokenSaver {
		if msgs, saved := compressMessagesRTK(req.Messages); saved > 0 {
			req.Messages = msgs
			log.Printf("flow_router RTK token saver: trimmed %d chars from tool results", saved)
		}
	}

	if settings != nil && settings.CavemanLevel != "" {
		injectCavemanIntoRequest(&req, settings.CavemanLevel)
	}

	preprocessToolCalls(&req)

	var brainInfo *brainEnrichInfo
	heavyEnriched := false
	if !isCrewLightModel(req.Model) {

		maybeInjectConstitution(ctx, &req, settings)

		brainInfo = maybeEnrichBrain(ctx, &req, settings)
		heavyEnriched = true
	}

	maybeInjectAntibodies(ctx, &req, settings)

	maybeInjectInstinct(ctx, &req, settings)

	req = maybeFilterTools(ctx, req, settings)

	req = applyInjectShaper(ctx, req, settings)

	resolvedModel, pinnedProvider := resolveModel(d, req.Model)
	req.Model = resolvedModel

	var comboFallback []string
	if pinnedProvider == "" {
		if combo, _ := store.GetComboByName(d, req.Model); combo != nil && len(combo.Models) > 0 {
			picked := pickComboModel(combo)
			log.Printf("flow_router combo %q (%s) → model %q", combo.Name, combo.Strategy, picked)
			req.Model = picked
			comboFallback = comboFallbackOrder(combo, picked)
		}
	}

	modelsToTry := append([]string{req.Model}, comboFallback...)
	nPrimary := len(modelsToTry)
	if settings == nil || settings.FallbackStrategy != "none" {
		modelsToTry = append(modelsToTry, globalFallbackModels(d, modelsToTry)...)
	}
	originalModel := modelsToTry[0]
	var lastModelErr error
	var lastModelStatus int
	for modelIdx, candidateModel := range modelsToTry {
		req.Model = candidateModel
		pin := pinnedProvider
		switch {
		case modelIdx >= nPrimary:

			rm, rp := resolveModel(d, candidateModel)
			req.Model, pin = rm, rp
			if !heavyEnriched && !isCrewLightModel(req.Model) {
				maybeInjectConstitution(ctx, &req, settings)
				brainInfo = maybeEnrichBrain(ctx, &req, settings)
				heavyEnriched = true
			}
			log.Printf("flow_router priority fallback: %q unavailable → trying %q (next ON provider)", originalModel, req.Model)
		case modelIdx > 0:
			log.Printf("flow_router combo per-model fallback: trying %q", candidateModel)
		}
		resp, status, err := dispatchSingleModel(ctx, d, req, settings, brainInfo, pin)
		if err == nil && resp != nil {
			return resp, status, nil
		}
		lastModelErr = err
		lastModelStatus = status

		stop := shouldStopComboFallback(status)

		if stop && modelIdx+1 < len(modelsToTry) && modelIdx+1 >= nPrimary && availabilityFailure(status) {
			stop = false
		}
		if stop {
			break
		}
	}
	return nil, lastModelStatus, lastModelErr
}

func availabilityFailure(status int) bool {
	switch status {
	case http.StatusNotFound, http.StatusRequestTimeout,
		http.StatusTooManyRequests, http.StatusPaymentRequired:
		return true
	}
	return status >= 500
}

func globalFallbackModels(d *sql.DB, tried []string) []string {
	provs, err := store.ListProviders(d)
	if err != nil {
		return nil
	}
	skip := make(map[string]bool, len(tried))
	for _, m := range tried {
		skip[strings.ToLower(strings.TrimSpace(m))] = true
	}
	var out []string
	for _, p := range provs {
		if !p.IsActive {
			continue
		}
		models, _ := p.Data[store.CfgModels].([]any)
		for _, mm := range models {
			ms, ok := mm.(string)
			if !ok {
				continue
			}
			ms = strings.TrimSpace(ms)
			if ms == "" || ms == "*" || strings.HasSuffix(ms, "*") {
				continue
			}
			key := strings.ToLower(ms)
			if skip[key] {
				continue
			}
			skip[key] = true
			out = append(out, ms)
			break
		}
	}
	return out
}

func dispatchSingleModel(ctx context.Context, d *sql.DB, req OpenAIRequest, settings *store.Settings, brainInfo *brainEnrichInfo, pinnedProvider string) (*OpenAIResponse, int, error) {
	matches, err := store.FindActiveByModel(d, req.Model)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("find provider: %w", err)
	}
	if pinnedProvider != "" {
		matches = pinProvider(d, matches, pinnedProvider)
	}
	if len(matches) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("no active provider supports model %q", req.Model)
	}

	if matches = filterDisabled(d, matches, req.Model); len(matches) == 0 {
		return nil, http.StatusForbidden, fmt.Errorf("model %q is disabled", req.Model)
	}

	keyID := apiKeyID(ctx)
	if key := APIKeyFromContext(ctx); key != nil {
		matches = filterByAllowedProviders(matches, key)
		if len(matches) == 0 {
			return nil, http.StatusForbidden, fmt.Errorf("api key %q not permitted for any provider serving model %q", key.Name, req.Model)
		}
	}

	if settings != nil && settings.IntentRouting.Enabled && promptIsPrivate(req, settings.IntentRouting.PrivatePatterns) {
		tag := settings.IntentRouting.PrivateTag
		if tag == "" {
			tag = "local"
		}
		local := filterByTag(matches, tag)
		if len(local) == 0 {
			return nil, http.StatusForbidden, fmt.Errorf("private prompt: no provider tagged %q available — refusing to route to cloud", tag)
		}
		matches = local
		log.Printf("flow_router intent-routing: private prompt → %d provider(s) tagged %q", len(local), tag)
	}

	if settings != nil && settings.CostRouting.Enabled {
		if !(settings.CostRouting.HonorExplicitModel && hasActiveProviderForModel(matches, req.Model)) {
			tier := ClassifyCost(req, settings.CostRouting)
			if tiered := filterByTier(matches, tier); len(tiered) > 0 {
				matches = tiered
				log.Printf("flow_router cost-routing: tier=%s → %d provider(s)", tier, len(tiered))
			}
		}
	}

	if settings != nil {
		matches = applyFallbackStrategy(matches, settings.FallbackStrategy, req.Model)
	}

	matches = reorderByModelLock(matches, req.Model)

	var lastErr error
	startTotal := time.Now()
	for _, p := range matches {

		var resp *OpenAIResponse
		var status int
		var err error
		for attempt := 0; ; attempt++ {
			start := time.Now()
			resp, status, err = forwardToProvider(ctx, &p, req)
			latencyMs := time.Since(start).Milliseconds()

			lr, ls, le := resp, status, err
			safego.GoLabel("logUsage", func() {
				logUsage(d, keyID, p.ID, req.Model, lr, ls, le, latencyMs)
			})
			// AUTO-REM (sadar-kuota): kalau kuota 5-jam langganan udah MEPET (>=95%/surpassed),
			// JANGAN retry-storm 429 — langsung break → lompat fallback (haiku/lokal). Nge-hammer
			// tembok yg ga bakal gerak cuma ngabisin window. State dari ratelimit_track_ext.go.
			if status != http.StatusTooManyRequests || attempt >= maxRateLimitRetries() || SubscriptionNearLimit() {
				if status == http.StatusTooManyRequests && SubscriptionNearLimit() {
					log.Printf("flow_router THROTTLE: kuota 5-jam mepet → skip retry %s, lompat fallback", p.Name)
				}
				break
			}
			wait := backoffDuration(attempt)
			log.Printf("flow_router 429 (rate-limit) model=%s provider=%s → antri %v lalu retry (%d/%d)",
				req.Model, p.Name, wait, attempt+1, maxRateLimitRetries())
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, http.StatusGatewayTimeout, ctx.Err()
			}
		}

		if err == nil && resp != nil {
			recoverTextToolCalls(resp)
			clearModelLock(p.ID, req.Model)
			log.Printf("flow_router dispatch model=%s → provider=%s tokens=%d",
				req.Model, p.Name, resp.Usage.TotalTokens)
			recordBrainContribution(d, settings, brainInfo, answerText(resp))

			respCopy := resp
			settingsCopy := settings
			safego.GoLabel("antibodyFeedback", func() {
				maybeReinforceAntibody(context.Background(), respCopy, settingsCopy)
			})
			return resp, status, nil
		}
		lastErr = err
		errText := ""
		if err != nil {
			errText = err.Error()
		}

		if status != http.StatusTooManyRequests {
			lockModel(p.ID, req.Model, status, errText)
		}
		log.Printf("flow_router fallback model=%s provider=%s failed (%v), trying next", req.Model, p.Name, err)
	}

	log.Printf("flow_router ALL providers exhausted model=%s total=%dms", req.Model, time.Since(startTotal).Milliseconds())
	return nil, http.StatusBadGateway, fmt.Errorf("all providers failed; last error: %w", lastErr)
}

// afterAnthropicResponse — POLA B hook (default no-op). Dipanggil tiap response Anthropic
// (termasuk 429) dengan header-nya → sibling ratelimit_track_ext.go (non-frozen) override
// buat baca anthropic-ratelimit-unified-* + simpan utilisasi 5h/7d di 1 STATE SHARE (semua
// agent lewat sini, nol duplikat). Default no-op = self-sufficient (hapus sibling → aman).
var afterAnthropicResponse = func(http.Header) {}

func forwardToProvider(ctx context.Context, p *store.ProviderConnection, req OpenAIRequest) (*OpenAIResponse, int, error) {

	acquireDispatchSlot()
	defer releaseDispatchSlot()

	format, _ := p.Data[store.CfgFormat].(string)
	baseURL, _ := p.Data[store.CfgBaseURL].(string)

	if format == "" {
		if resolved := providercompat.ResolveFormat(p.Provider); resolved != "" {
			format = resolved
		}
	}
	if baseURL == "" {
		baseURL = providercompat.ResolveBaseURL(p.Provider, baseURL)
	}

	if baseURL == "" {
		return nil, 0, fmt.Errorf("provider %s missing baseUrl", p.ID)
	}

	defer wakeLocalIfNeeded(baseURL)()

	if ex := executors.Get(format); ex != nil {
		body, u, st, err := ex.NonStream(ctx, p, executorRequest(req))
		if err != nil {
			return nil, st, err
		}
		var resp OpenAIResponse
		if jerr := json.Unmarshal(body, &resp); jerr != nil {
			return nil, http.StatusBadGateway, fmt.Errorf("executor %s decode: %w", format, jerr)
		}
		if resp.Usage.TotalTokens == 0 {
			resp.Usage.PromptTokens = u.PromptTokens
			resp.Usage.CompletionTokens = u.CompletionTokens
			resp.Usage.TotalTokens = u.TotalTokens
		}
		return &resp, st, nil
	}

	switch format {
	case "anthropic":
		return forwardAnthropic(ctx, p, baseURL, req)
	case "openai", "":
		return forwardOpenAICompat(ctx, p, baseURL, req)
	case "gemini":
		return forwardGemini(ctx, p, baseURL, req)
	default:
		return nil, 0, fmt.Errorf("unknown format: %s", format)
	}
}

func forwardOpenAICompat(ctx context.Context, p *store.ProviderConnection, baseURL string, req OpenAIRequest) (*OpenAIResponse, int, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build req: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
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
		return nil, resp.StatusCode, fmt.Errorf("upstream %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var out OpenAIResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("parse resp: %w", err)
	}
	return &out, http.StatusOK, nil
}

func forwardAnthropic(ctx context.Context, p *store.ProviderConnection, baseURL string, req OpenAIRequest) (*OpenAIResponse, int, error) {
	if hasToolContext(req) {
		return forwardAnthropicWithTools(ctx, p, baseURL, req)
	}

	anthrReq := AnthropicRequest{
		Model:       normalizeClaudeModel(req.Model),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	if anthrReq.MaxTokens <= 0 {
		anthrReq.MaxTokens = 4096
	}
	var sysParts []string
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			sysParts = append(sysParts, m.Content)
		case "user", "assistant":
			anthrReq.Messages = append(anthrReq.Messages, AnthropicMessage{Role: m.Role, Content: m.Content})
		}
	}
	if len(sysParts) > 0 {
		anthrReq.System = strings.Join(sysParts, "\n\n")
	}

	body, err := json.Marshal(anthrReq)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal anthropic: %w", err)
	}

	if claudeUsesOAuth(p) {
		body = applyClaudeIdentityCloak(body, "")
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build req: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	// UA cloak: samain PERSIS sama Claude Code asli (claude-code/<versi>) — BUKAN
	// 'claude-cli/1.0.0' yg bisa ke-flag tier rate-limit beda. Versi via FLOWORK_CLOAK_VERSION.
	httpReq.Header.Set("User-Agent", "claude-code/"+claudeVersion())
	if err := applyAuth(httpReq, p); err != nil {
		return nil, http.StatusUnauthorized, err
	}

	resp, err := outboundClient(ctx).Do(httpReq)
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("upstream: %w", err)
	}
	defer resp.Body.Close()
	afterAnthropicResponse(resp.Header) // SADAR-KUOTA: track anthropic-ratelimit-* (incl 429)

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("anthropic %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var anthrResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthrResp); err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("parse anthropic: %w", err)
	}

	var content string
	for _, c := range anthrResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}
	stopReason := "stop"
	switch anthrResp.StopReason {
	case "end_turn", "stop_sequence":
		stopReason = "stop"
	case "max_tokens":
		stopReason = "length"
	case "tool_use":
		stopReason = "tool_calls"
	}
	return &OpenAIResponse{
		ID:      anthrResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []OpenAIChoice{{
			Index:        0,
			Message:      OpenAIMessage{Role: "assistant", Content: content},
			FinishReason: stopReason,
		}},
		Usage: OpenAIUsage{
			PromptTokens:     anthrResp.Usage.InputTokens,
			CompletionTokens: anthrResp.Usage.OutputTokens,
			TotalTokens:      anthrResp.Usage.InputTokens + anthrResp.Usage.OutputTokens,
		},
	}, http.StatusOK, nil
}

func applyAuth(req *http.Request, p *store.ProviderConnection) error {
	switch p.AuthType {
	case store.AuthTypeNone:
		return nil
	case store.AuthTypeAPIKey:
		k, _ := p.Data[store.CfgAPIKey].(string)
		if k == "" {
			return fmt.Errorf("provider %s missing apiKey", p.ID)
		}

		if p.Provider == "anthropic" {
			req.Header.Set("x-api-key", k)
		} else {
			req.Header.Set("Authorization", "Bearer "+k)
		}
		return nil
	case store.AuthTypeSubscription:

		src, _ := p.Data[store.CfgTokenSource].(string)
		switch src {
		case "claude_credentials":

			c, err := creds.LoadValid()
			if err != nil {
				return fmt.Errorf("claude creds: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+c.ClaudeAiOauth.AccessToken)
			return nil
		case "codex_auth":
			tok, err := creds.LoadCodexToken()
			if err != nil {
				return fmt.Errorf("codex auth: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+tok)
			return nil
		case "cursor_session":
			tok, err := creds.LoadCursorToken()
			if err != nil {
				return fmt.Errorf("cursor auth: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+tok)
			return nil
		default:
			return fmt.Errorf("unknown subscription tokenSource: %s", src)
		}
	}
	return fmt.Errorf("unknown authType: %s", p.AuthType)
}

func shouldStopComboFallback(status int) bool {
	return status > 0 && status < 500 && status != http.StatusNotFound
}

func comboFallbackOrder(c *store.Combo, picked string) []string {
	if c == nil || len(c.Models) < 2 {
		return nil
	}
	out := make([]string, 0, len(c.Models)-1)
	for _, m := range c.Models {
		if m == picked {
			continue
		}
		out = append(out, m)
	}
	return out
}

func pickComboModel(c *store.Combo) string {
	if len(c.Models) == 0 {
		return ""
	}
	switch c.Strategy {
	case store.ComboStrategyRoundRobin:
		i := nextRoundRobin("combo:"+c.ID, len(c.Models))
		return c.Models[i]
	case store.ComboStrategyRandom:

		return c.Models[int(uint64(time.Now().UnixNano())%uint64(len(c.Models)))]
	case store.ComboStrategyCostOptimal:

		bestModel := c.Models[0]
		bestCost := estimateCost(bestModel, 1000, 1000)
		for _, m := range c.Models[1:] {
			cost := estimateCost(m, 1000, 1000)
			if cost > 0 && (bestCost == 0 || cost < bestCost) {
				bestModel = m
				bestCost = cost
			}
		}
		return bestModel
	default:
		return c.Models[0]
	}
}

func normalizeClaudeModel(m string) string {
	m = strings.TrimSpace(m)
	for _, prefix := range []string{"cc/", "anthropic/", "claude/"} {
		m = strings.TrimPrefix(m, prefix)
	}
	if m == "" {
		return "claude-haiku-4-5"
	}
	return m
}

func logUsage(d any, apiKeyID, providerID, model string, resp *OpenAIResponse, status int, errIn error, latencyMs int64) {
	db, ok := d.(*sql.DB)
	if !ok || db == nil {
		return
	}
	entry := &store.LogEntry{
		APIKeyID:   apiKeyID,
		ProviderID: providerID,
		Model:      model,
		StatusCode: status,
		LatencyMs:  latencyMs,
	}
	if errIn != nil {
		entry.Error = errIn.Error()
	}
	if resp != nil {
		entry.PromptTokens = resp.Usage.PromptTokens
		entry.CompletionTokens = resp.Usage.CompletionTokens
		entry.TotalTokens = resp.Usage.TotalTokens
		entry.CostUsd = estimateCost(model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	}
	_ = store.LogRequest(db, entry)
}

func estimateCost(model string, promptTok, complTok int) float64 {
	d, err := store.Open()
	if err != nil {
		return 0
	}
	pr, err := store.LookupPricingByModel(d, model)
	if err != nil || pr == nil {
		return 0
	}
	return (float64(promptTok)/1e6)*pr.InputUsdPer1M + (float64(complTok)/1e6)*pr.OutputUsdPer1M
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
