// === LOCKED FILE (soft) === Status: STABLE (owner-approved 2026-06-15). Tested:
// session CRUD + send (architect + group modes) + auto-title + persistence verified.
//
// chat_sessions.go — HTTP API for the persistent "Chat" tab (Group section). Sessions
// + messages live in flowork.db (chatdb.go) so they survive a PC shutdown and keep
// full context. A session targets either a GROUP (talk to a team) or the ARCHITECT
// (brainstorm + build teams). "How Claude works": each send replays the FULL history
// to the target (big model holds it). Loopback-only (allowlisted in floworkauth).
//
//	GET  /api/chat/sessions              → list sessions
//	POST /api/chat/sessions              → create {title?,mode,target_id?,model?}
//	POST /api/chat/sessions/rename?id=   → {title}
//	POST /api/chat/sessions/delete?id=
//	POST /api/chat/sessions/meta?id=     → {mode?,target_id,model}  (switch target/model)
//	GET  /api/chat/sessions/messages?id= → full message list
//	POST /api/chat/send                  → {session_id,text,images?} → reply (and persist both turns)
//	                                       images = data URL base64 (multimodal paste, chat_vision.go)
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernelhost"
)

func newSessionID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "s" + time.Now().Format("20060102150405")
	}
	return "s_" + hex.EncodeToString(b)
}

// snippetTitle — a short title from the first user message.
func snippetTitle(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if s == "" {
		return "New chat"
	}
	r := []rune(s)
	if len(r) > 48 {
		return string(r[:48]) + "…"
	}
	return string(r)
}

// chatSessionsHandler — GET list / POST create.
func chatSessionsHandler(store *floworkdb.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			list, err := store.ListChatSessions()
			if err != nil {
				tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			tfWriteJSON(w, 0, map[string]any{"sessions": list})
		case http.MethodPost:
			var body struct {
				Title    string `json:"title"`
				Mode     string `json:"mode"`
				TargetID string `json:"target_id"`
				Model    string `json:"model"`
			}
			_ = json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
			cs := floworkdb.ChatSession{
				ID: newSessionID(), Title: strings.TrimSpace(body.Title),
				Mode: strings.TrimSpace(body.Mode), TargetID: strings.TrimSpace(body.TargetID),
				Model: strings.TrimSpace(body.Model),
			}
			if err := store.CreateChatSession(cs); err != nil {
				tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			got, _ := store.GetChatSession(cs.ID)
			tfWriteJSON(w, 0, map[string]any{"ok": true, "session": got})
		default:
			tfWriteJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "GET or POST"})
		}
	}
}

func chatSessionRenameHandler(store *floworkdb.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		var body struct {
			Title string `json:"title"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
		title := strings.TrimSpace(body.Title)
		if id == "" || title == "" {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "id + title required"})
			return
		}
		if err := store.RenameChatSession(id, title); err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		tfWriteJSON(w, 0, map[string]any{"ok": true})
	}
}

func chatSessionDeleteHandler(store *floworkdb.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "id required"})
			return
		}
		if err := store.DeleteChatSession(id); err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		tfWriteJSON(w, 0, map[string]any{"ok": true})
	}
}

func chatSessionMetaHandler(store *floworkdb.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "id required"})
			return
		}
		var body struct {
			Mode     string `json:"mode"`
			TargetID string `json:"target_id"`
			Model    string `json:"model"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
		if err := store.UpdateChatSessionMeta(id, strings.TrimSpace(body.Mode), strings.TrimSpace(body.TargetID), strings.TrimSpace(body.Model)); err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		tfWriteJSON(w, 0, map[string]any{"ok": true})
	}
}

func chatMessagesHandler(store *floworkdb.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "id required"})
			return
		}
		msgs, err := store.ListChatMessages(id, 0)
		if err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		tfWriteJSON(w, 0, map[string]any{"messages": msgs})
	}
}

// buildGroupTranscript — render the conversation into a single context+question text
// for a GROUP coordinator (which takes one message). Full history → context, last user
// turn → the live question.
func buildGroupTranscript(history []floworkdb.ChatMessage) string {
	if len(history) == 0 {
		return ""
	}
	// Jalur GROUP text-only: lampiran gambar ditandai teks (vision penuh = jalur architect).
	imgNote := func(m floworkdb.ChatMessage) string {
		if len(m.Images) == 0 {
			return ""
		}
		return " [📷 user melampirkan gambar — jalur tim belum dukung vision]"
	}
	last := history[len(history)-1].Content + imgNote(history[len(history)-1])
	if len(history) == 1 {
		return last
	}
	var b strings.Builder
	b.WriteString("Konteks percakapan sebelumnya:\n")
	for _, m := range history[:len(history)-1] {
		who := "User"
		if m.Role == "assistant" {
			who = "Tim"
		}
		b.WriteString(who)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString(imgNote(m))
		b.WriteString("\n")
	}
	b.WriteString("\nPertanyaan/permintaan terbaru User: ")
	b.WriteString(last)
	return b.String()
}

// chatSendHandler — append the user turn, run the target (architect brain or group),
// persist + return the assistant turn. Both turns are saved → full memory next time.
func chatSendHandler(host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			tfWriteJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST only"})
			return
		}
		var body struct {
			SessionID string   `json:"session_id"`
			Text      string   `json:"text"`
			Images    []string `json:"images"` // data URL base64 (multimodal paste)
		}
		// 16MB: gambar base64 gede (limit per-gambar/total di validChatImages).
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<24)).Decode(&body); err != nil {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
			return
		}
		text := strings.TrimSpace(body.Text)
		images, ierr := validChatImages(body.Images)
		if ierr != nil {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": ierr.Error()})
			return
		}
		if body.SessionID == "" || (text == "" && len(images) == 0) {
			tfWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "session_id + text (atau gambar) required"})
			return
		}
		sess, err := store.GetChatSession(body.SessionID)
		if err != nil {
			tfWriteJSON(w, http.StatusNotFound, map[string]any{"error": "session not found"})
			return
		}
		// Persist the user turn first (images ikut kesimpan → history vision utuh).
		if _, err := store.AddChatMessageImages(sess.ID, "user", text, images); err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		// Auto-title from the first user message.
		if strings.TrimSpace(sess.Title) == "" || sess.Title == "New chat" {
			title := snippetTitle(text)
			if text == "" {
				title = "📷 Gambar"
			}
			_ = store.RenameChatSession(sess.ID, title)
		}
		history, err := store.ListChatMessages(sess.ID, 0)
		if err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 290*time.Second)
		defer cancel()

		var reply string
		if sess.Mode == "group" && strings.TrimSpace(sess.TargetID) != "" {
			raw, ierr := host.InvokeAgentMessage(ctx, sess.TargetID, buildGroupTranscript(history), "cli:owner")
			if ierr != nil {
				tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": ierr.Error()})
				return
			}
			reply = extractReply(raw)
		} else {
			reply, err = architectChat(ctx, host, store, groups, history, sess.Model)
			if err != nil {
				tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
		}
		if strings.TrimSpace(reply) == "" {
			reply = "(kosong)"
		}
		if _, err := store.AddChatMessage(sess.ID, "assistant", reply); err != nil {
			tfWriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		tfWriteJSON(w, 0, map[string]any{"reply": reply})
	}
}

// extractReply — pull the human text from an agent's raw emit ({"reply":...} or raw).
func extractReply(raw string) string {
	reply := strings.TrimSpace(raw)
	var emitted map[string]any
	if json.Unmarshal([]byte(raw), &emitted) == nil {
		if rv, ok := emitted["reply"].(string); ok {
			return rv
		}
		if ev, ok := emitted["error"].(string); ok {
			return "⚠️ " + ev
		}
	}
	return reply
}
