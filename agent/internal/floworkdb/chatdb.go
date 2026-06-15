// chatdb.go — PERSISTENT CHAT SESSIONS (flowork.db). Backs the GUI "Chat" tab in
// the Group section: ChatGPT-style sessions that survive a PC shutdown and let the
// agent remember context. A session targets either a GROUP (mode=group, talk to a
// team) or the ARCHITECT (mode=architect, brainstorm + build teams/agents). The
// server owns the conversation (source of truth) and feeds a condensed history back
// to the target on each turn — small prompts (ant principle), local-LLM friendly.
//
//	chat_session  — one conversation (title, mode, target, model).
//	chat_message  — one turn (role user|assistant, content).
//
// Owner-local: these tables live in the owner-level flowork.db, edited only via the
// loopback-gated /api/chat/sessions* endpoints.

package floworkdb

// ChatSession — one persisted conversation.
type ChatSession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Mode      string `json:"mode"`      // "group" | "architect"
	TargetID  string `json:"target_id"` // group id when mode=group ("" for architect)
	Model     string `json:"model"`     // "" = use the target's/agent's default
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ChatMessage — one turn in a session.
type ChatMessage struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"` // "user" | "assistant"
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// EnsureChatSchema — create the chat tables (idempotent). Called at boot.
func (s *Store) EnsureChatSchema() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS chat_session (
		id         TEXT PRIMARY KEY,
		title      TEXT NOT NULL DEFAULT 'New chat',
		mode       TEXT NOT NULL DEFAULT 'architect',
		target_id  TEXT NOT NULL DEFAULT '',
		model      TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS chat_message (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role       TEXT NOT NULL,
		content    TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return err
	}
	_, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_message_session ON chat_message(session_id, id)`)
	return err
}

// CreateChatSession — insert a new session (caller supplies the id). Blank fields
// fall back to sensible defaults.
func (s *Store) CreateChatSession(cs ChatSession) error {
	if cs.Title == "" {
		cs.Title = "New chat"
	}
	if cs.Mode != "group" {
		cs.Mode = "architect"
	}
	_, err := s.db.Exec(
		`INSERT INTO chat_session(id,title,mode,target_id,model) VALUES(?,?,?,?,?)`,
		cs.ID, cs.Title, cs.Mode, cs.TargetID, cs.Model)
	return err
}

// ListChatSessions — every session, most-recently-updated first.
func (s *Store) ListChatSessions() ([]ChatSession, error) {
	rows, err := s.db.Query(
		`SELECT id,title,mode,target_id,model,created_at,updated_at FROM chat_session ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatSession{}
	for rows.Next() {
		var c ChatSession
		if err := rows.Scan(&c.ID, &c.Title, &c.Mode, &c.TargetID, &c.Model, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChatSession — one session by id.
func (s *Store) GetChatSession(id string) (ChatSession, error) {
	var c ChatSession
	err := s.db.QueryRow(
		`SELECT id,title,mode,target_id,model,created_at,updated_at FROM chat_session WHERE id=?`, id).
		Scan(&c.ID, &c.Title, &c.Mode, &c.TargetID, &c.Model, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// RenameChatSession — set the title.
func (s *Store) RenameChatSession(id, title string) error {
	_, err := s.db.Exec(`UPDATE chat_session SET title=?, updated_at=datetime('now') WHERE id=?`, title, id)
	return err
}

// UpdateChatSessionMeta — change the target/model of a session (e.g. switch which
// group it talks to). Empty mode keeps the existing one.
func (s *Store) UpdateChatSessionMeta(id, mode, targetID, model string) error {
	if mode == "" {
		_, err := s.db.Exec(`UPDATE chat_session SET target_id=?, model=?, updated_at=datetime('now') WHERE id=?`, targetID, model, id)
		return err
	}
	_, err := s.db.Exec(`UPDATE chat_session SET mode=?, target_id=?, model=?, updated_at=datetime('now') WHERE id=?`, mode, targetID, model, id)
	return err
}

// DeleteChatSession — remove a session and all its messages.
func (s *Store) DeleteChatSession(id string) error {
	if _, err := s.db.Exec(`DELETE FROM chat_message WHERE session_id=?`, id); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM chat_session WHERE id=?`, id)
	return err
}

// AddChatMessage — append a turn and bump the session's updated_at.
func (s *Store) AddChatMessage(sessionID, role, content string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO chat_message(session_id,role,content) VALUES(?,?,?)`, sessionID, role, content)
	if err != nil {
		return 0, err
	}
	_, _ = s.db.Exec(`UPDATE chat_session SET updated_at=datetime('now') WHERE id=?`, sessionID)
	return res.LastInsertId()
}

// ListChatMessages — a session's turns oldest-first. limit<=0 → all.
func (s *Store) ListChatMessages(sessionID string, limit int) ([]ChatMessage, error) {
	q := `SELECT id,session_id,role,content,created_at FROM chat_message WHERE session_id=? ORDER BY id ASC`
	args := []any{sessionID}
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RecentChatMessages — the last N turns of a session, returned oldest-first (for
// feeding condensed history back to the target without blowing the prompt).
func (s *Store) RecentChatMessages(sessionID string, n int) ([]ChatMessage, error) {
	if n <= 0 {
		n = 12
	}
	rows, err := s.db.Query(
		`SELECT id,session_id,role,content,created_at FROM chat_message WHERE session_id=? ORDER BY id DESC LIMIT ?`,
		sessionID, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tmp := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		tmp = append(tmp, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// reverse → oldest-first
	for i, j := 0, len(tmp)-1; i < j; i, j = i+1, j-1 {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	}
	return tmp, nil
}
