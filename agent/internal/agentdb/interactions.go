// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package agentdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Interaction struct {
	ID         int64          `json:"id"`
	Channel    string         `json:"channel"`
	Direction  string         `json:"direction"`
	Actor      string         `json:"actor"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata"`
	OccurredAt string         `json:"occurred_at"`
}

func (s *Store) LogInteraction(channel, direction, actor, content string, metadata map[string]any) (int64, error) {
	content = SanitizeText(content)   // seam scrub rahasia (sanitize_seam.go)
	metadata = SanitizeMeta(metadata) // no-op kalau ext ga kepasang
	s.mu.Lock()
	defer s.mu.Unlock()

	if channel == "" || direction == "" {
		return 0, fmt.Errorf("channel + direction required")
	}
	if direction != "in" && direction != "out" {
		return 0, fmt.Errorf("direction must be 'in' or 'out'")
	}

	const maxContentBytes = 8 * 1024
	if len(content) > maxContentBytes {
		content = content[:maxContentBytes] + "…[truncated]"
	}

	var metaJSON string
	if len(metadata) > 0 {
		b, err := json.Marshal(metadata)
		if err == nil {
			metaJSON = string(b)
		}
	}
	if metaJSON == "" {
		metaJSON = "{}"
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO interactions(channel, direction, actor, content, metadata, occurred_at) VALUES(?, ?, ?, ?, ?, ?)`,
		channel, direction, actor, content, metaJSON, ts,
	)
	if err != nil {
		return 0, fmt.Errorf("insert interaction: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) ListInteractions(channel, actor string, limit int) ([]Interaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 500 {
		limit = 50
	}

	query := `SELECT id, channel, direction, actor, content, metadata, occurred_at
	          FROM interactions WHERE deleted_at IS NULL`
	args := []any{}
	if channel != "" {
		query += ` AND channel = ?`
		args = append(args, channel)
	}
	if actor != "" {
		query += ` AND actor = ?`
		args = append(args, actor)
	}
	query += ` ORDER BY occurred_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query interactions: %w", err)
	}
	defer rows.Close()

	var out []Interaction
	for rows.Next() {
		var it Interaction
		var metaRaw string
		if err := rows.Scan(&it.ID, &it.Channel, &it.Direction, &it.Actor, &it.Content, &metaRaw, &it.OccurredAt); err != nil {
			return nil, err
		}
		if metaRaw != "" && metaRaw != "{}" {
			_ = json.Unmarshal([]byte(metaRaw), &it.Metadata)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) PruneInteractions(olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE interactions SET deleted_at = CURRENT_TIMESTAMP
		 WHERE deleted_at IS NULL AND occurred_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("prune interactions: %w", err)
	}
	return res.RowsAffected()
}

func (s *Store) CountInteractions() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var n int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM interactions WHERE deleted_at IS NULL`).Scan(&n); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}
