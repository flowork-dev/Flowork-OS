// FROZEN brain-core — desain abadi Flowork. Kalau ini bikin lo "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30
// Reason: Audit pass — Brain drawer/embeddings/skills.
// Update 2026-06-16 (owner-approved #6): auto-context agent pakai brain.SemanticRetrieve (by-MAKNA)
// ganti Retrieve (FTS). Re-locked.
// 2026-06-13 (release audit → test → re-lock): added MaxSkillBodyChars cap to buildBrainSystem
//   (capSkillBody, default 0 = legacy no-cap). PROVEN: dumping big external skills (24×~11KB) sent
//   the commander prompt 2791→10764 tok (+286%); cap=700 dropped it to 756 (−93%). Lets big skills
//   load cheaply. Settings field Brain.MaxSkillBodyChars; set via PUT /api/brain/config.

// Brain enrichment (the shared-brain core).

package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/brain"
	"github.com/flowork-os/flowork_Router/internal/store"
)

// enrichRetrieve — ⭐ SEAM (Rule 7 POLA B, 2026-07-02 owner-approved): sumber snippet
// enrichment. Default = perilaku lama (SemanticRetrieve, skor dinormalisasi ke top-hit).
// Override via sibling non-frozen enrich_selective_ext.go (lantai relevansi absolut,
// switch GUI FLOWORK_ENRICH_MINSCORE) TANPA buka file frozen ini. Sibling dihapus →
// balik default aman. 📄 Dok: FLowork_os/lock/prompt-diet.md
var enrichRetrieve = func(ctx context.Context, db *sql.DB, query string, opts brain.RetrieveOpts) ([]brain.Snippet, error) {
	return brain.SemanticRetrieve(ctx, db, query, opts)
}

// brainEnrichInfo carries what was injected, so the dispatcher can record the
// interaction for compounding after the answer is known. nil = not enriched.
type brainEnrichInfo struct {
	Query   string
	Mode    string
	Agent   string
	Sources []brain.Snippet
}

// maybeEnrichBrain turns an ordinary request into a brain-backed one: when the
// requested model matches settings.Brain.Model, it retrieves relevant knowledge
// from the Memory Palace + selects applicable skills and injects them as a
// system message. This is what makes ANY OpenAI-compatible agent (OpenClaw,
// Hermes, Cursor, flowork) share the same intelligence — the enrichment happens
// server-side, transparently.
// It mutates req.Messages in place and returns enrichment info (nil if not
// applied). Fails open: any missing piece (disabled, no DB, empty query, no
// hits) simply skips enrichment so the request still flows through normal
// dispatch.
// The request model is left unchanged, so the existing alias/provider/fallback
// machinery still resolves Brain.Model to a real backend (Ollama or cloud).
func maybeEnrichBrain(ctx context.Context, req *OpenAIRequest, settings *store.Settings) *brainEnrichInfo {
	if settings == nil || !settings.Brain.Enabled {
		return nil
	}
	trigger := settings.Brain.Model
	if trigger == "" {
		trigger = "flowork-brain"
	}
	// AlwaysOn = brain enrichment fires for every request, regardless of the
	// model the client picked. Without it, only requests that explicitly
	// name `Brain.Model` reach the doctrine.
	if !settings.Brain.AlwaysOn && !strings.EqualFold(strings.TrimSpace(req.Model), trigger) {
		return nil
	}

	if settings.Brain.DBPath != "" {
		brain.SetDBPath(settings.Brain.DBPath)
	}
	if !brain.Available() {
		log.Printf("flow_router brain: enabled but no DB at %q — skipping enrichment", brain.DBPath())
		return nil
	}
	db, err := brain.Open()
	if err != nil {
		log.Printf("flow_router brain: open failed: %v — skipping enrichment", err)
		return nil
	}

	query := lastUserText(req.Messages)
	if query == "" {
		return nil
	}

	topK := settings.Brain.TopK
	if topK <= 0 {
		topK = 5
	}
	maxLen := settings.Brain.MaxSnippetChars
	if maxLen <= 0 {
		maxLen = 600
	}
	// #6 (owner 2026-06-16): auto-context agent pakai SemanticRetrieve (by-MAKNA) — grounding agent
	// jadi semantik, bukan keyword. Fallback FTS otomatis selama index belum jadi.
	snips, err := enrichRetrieve(ctx, db, query, brain.RetrieveOpts{
		Limit: topK, Wings: settings.Brain.Wings, MaxContentLen: maxLen,
	})
	if err != nil {
		log.Printf("flow_router brain: retrieve error: %v — skipping enrichment", err)
		return nil
	}

	var skills []brain.SkillDoc
	if settings.Brain.Skills {
		n := settings.Brain.SkillTopK
		if n <= 0 {
			n = 2
		}
		skills = brain.SelectSkills(query, n)
	}

	if len(snips) == 0 && len(skills) == 0 {
		return nil // nothing to add — let the model answer plain
	}

	sysMsg := buildBrainSystem(snips, skills, settings.Brain.Mode, settings.Brain.MaxSkillBodyChars)
	req.Messages = injectSystem(req.Messages, sysMsg, settings.Brain.Mode)
	mode := modeOrDefault(settings.Brain.Mode)
	log.Printf("flow_router brain: enriched model=%q mode=%s (%d snippets, %d skills)",
		req.Model, mode, len(snips), len(skills))
	return &brainEnrichInfo{Query: query, Mode: mode, Agent: agentName(ctx), Sources: snips}
}

// agentName identifies the calling agent for contribution attribution. Uses the
// inbound API key name when present, else "anonymous".
func agentName(ctx context.Context) string {
	if k := APIKeyFromContext(ctx); k != nil && k.Name != "" {
		return k.Name
	}
	return "anonymous"
}

// recordBrainContribution queues a brain interaction for the compounding loop.
// No-op unless settings.Brain.Record is on (default off → plug-and-play). The
// answer may be empty (e.g. streaming) — the query + retrieved sources are still
// a useful training signal. Best-effort: errors are logged, never fatal.
func recordBrainContribution(d *sql.DB, settings *store.Settings, info *brainEnrichInfo, answer string) {
	if info == nil || settings == nil || !settings.Brain.Record {
		return
	}
	srcJSON, _ := json.Marshal(info.Sources)
	if err := store.AddBrainContribution(d, store.BrainContribution{
		Agent:   info.Agent,
		Model:   settings.Brain.Model,
		Mode:    info.Mode,
		Query:   info.Query,
		Sources: string(srcJSON),
		Answer:  answer,
	}); err != nil {
		log.Printf("flow_router brain: record contribution failed: %v", err)
	}
}

// answerText extracts the assistant text from a completion response.
func answerText(resp *OpenAIResponse) string {
	if resp == nil || len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

// lastUserText returns the text content of the most recent user message.
func lastUserText(msgs []OpenAIMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return strings.TrimSpace(msgs[i].Content)
		}
	}
	return ""
}

// buildBrainSystem renders retrieved knowledge + skills into a system message.
// In "brain" mode it adds a persona preface so the model fully assumes the
// brain identity; in "augment" mode it stays neutral (additional context only).
func buildBrainSystem(snips []brain.Snippet, skills []brain.SkillDoc, mode string, maxSkillChars int) string {
	var b strings.Builder
	if modeOrDefault(mode) == "brain" {
		b.WriteString("You are operating with a shared knowledge brain. Use the knowledge and follow the skills below as your own expertise.\n\n")
	}
	if len(snips) > 0 {
		b.WriteString("## Relevant knowledge\n")
		b.WriteString("Use the following retrieved knowledge when it helps; ignore anything irrelevant. Do not fabricate sources.\n\n")
		for i, s := range snips {
			src := s.Wing
			if s.Room != "" {
				src += "/" + s.Room
			}
			fmt.Fprintf(&b, "### [%d] %s\n%s\n\n", i+1, src, strings.TrimSpace(s.Content))
		}
	}
	if len(skills) > 0 {
		b.WriteString("## Applicable skills\n")
		b.WriteString("Apply these working methods when relevant:\n\n")
		for _, sk := range skills {
			if sk.Description != "" {
				fmt.Fprintf(&b, "### %s — %s\n", sk.Name, sk.Description)
			} else {
				fmt.Fprintf(&b, "### %s\n", sk.Name)
			}
			fmt.Fprintf(&b, "%s\n\n", capSkillBody(strings.TrimSpace(sk.Body), maxSkillChars))
		}
	}
	return strings.TrimSpace(b.String())
}

// capSkillBody bounds an injected skill body to maxChars runes (UTF-8 safe). 0 = no cap (legacy
// behavior). Keeps big external skills (the Claude-Code-style 10-19KB ones) from blowing up the
// commander prompt — the actionable head of a skill carries most of the value; the long tail is
// reference. Set Brain.MaxSkillBodyChars to enable.
func capSkillBody(body string, maxChars int) string {
	if maxChars <= 0 {
		return body
	}
	r := []rune(body)
	if len(r) <= maxChars {
		return body
	}
	return string(r[:maxChars]) + " …[skill dipotong — head saja, doktrin cheap-smart]"
}

// injectSystem inserts the brain system message.
//   - augment: after any leading caller system messages (caller's prompt stays primary)
//   - brain:   at index 0 (brain identity dominates)
func injectSystem(msgs []OpenAIMessage, content, mode string) []OpenAIMessage {
	if content == "" {
		return msgs
	}
	sys := OpenAIMessage{Role: "system", Content: content}
	// CACHE BOUNDARY: pas prompt-cache ON, enrichment WAJIB masuk SETELAH blok system
	// stabil (persona) biar prefix stabil bisa di-cache Anthropic (prepend = prefix
	// batal tiap turn). Prepend (mode brain) cuma dipakai pas cache OFF (perilaku lama).
	if modeOrDefault(mode) == "brain" && !promptCacheEnabled() {
		return append([]OpenAIMessage{sys}, msgs...)
	}
	// augment / cache-on: find end of leading system block
	insertAt := 0
	for insertAt < len(msgs) && msgs[insertAt].Role == "system" {
		insertAt++
	}
	out := make([]OpenAIMessage, 0, len(msgs)+1)
	out = append(out, msgs[:insertAt]...)
	out = append(out, sys)
	out = append(out, msgs[insertAt:]...)
	return out
}

func modeOrDefault(mode string) string {
	if mode == "brain" {
		return "brain"
	}
	return "augment"
}
