// Owner: Mr.Dev · github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN brain-core — jangan edit tanpa unfreeze owner. Arsitektur & alasan: lihat lock/brain.md

package agentdb

import (
	"database/sql"
	"fmt"
	"time"
)

type Mistake struct {
	ID            int64  `json:"id"`
	Category      string `json:"category"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	ContextOrigin string `json:"context_origin"`
	Tier          string `json:"tier"`
	HitCount      int64  `json:"hit_count"`
	LastHitAt     string `json:"last_hit_at"`
	CreatedAt     string `json:"created_at"`
	PromotedAt    string `json:"promoted_at,omitempty"`
	PromotedToID  string `json:"promoted_to_id,omitempty"`
}

func (s *Store) AddMistake(category, title, content, contextOrigin string) (int64, bool, error) {
	title = SanitizeText(title) // seam scrub rahasia (sanitize_seam.go)
	content = SanitizeText(content)
	s.mu.Lock()
	defer s.mu.Unlock()

	if category == "" || title == "" || content == "" {
		return 0, false, fmt.Errorf("category + title + content required")
	}

	const (
		maxContentBytes = 4 * 1024
		maxTitleBytes   = 256
	)
	if len(content) > maxContentBytes {
		content = content[:maxContentBytes] + "…[truncated]"
	}
	if len(title) > maxTitleBytes {
		title = title[:maxTitleBytes] + "…"
	}

	ts := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return 0, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var existingID int64
	err = tx.QueryRow(
		`SELECT id FROM mistakes_local WHERE category = ? AND title = ?`,
		category, title,
	).Scan(&existingID)

	switch {
	case err == sql.ErrNoRows:

		res, ierr := tx.Exec(
			`INSERT INTO mistakes_local(category, title, content, context_origin, last_hit_at, created_at)
			 VALUES(?, ?, ?, ?, ?, ?)`,
			category, title, content, contextOrigin, ts, ts,
		)
		if ierr != nil {
			return 0, false, fmt.Errorf("insert mistake: %w", ierr)
		}
		newID, _ := res.LastInsertId()
		if cerr := tx.Commit(); cerr != nil {
			return 0, false, fmt.Errorf("commit insert: %w", cerr)
		}
		tx = nil
		return newID, true, nil

	case err != nil:
		return 0, false, fmt.Errorf("lookup mistake: %w", err)

	default:

		_, uerr := tx.Exec(
			`UPDATE mistakes_local SET
			    content     = ?,
			    last_hit_at = ?,
			    hit_count   = hit_count + 1,
			    deleted_at  = NULL,
			    deleted_by  = NULL
			 WHERE id = ?`,
			content, ts, existingID,
		)
		if uerr != nil {
			return 0, false, fmt.Errorf("upsert mistake: %w", uerr)
		}
		if cerr := tx.Commit(); cerr != nil {
			return 0, false, fmt.Errorf("commit upsert: %w", cerr)
		}
		tx = nil
		return existingID, false, nil
	}
}

func (s *Store) ListMistakes(tier string, limit int) ([]Mistake, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 500 {
		limit = 50
	}

	query := `SELECT id, category, title, content, context_origin, tier, hit_count,
	                 last_hit_at, created_at,
	                 COALESCE(promoted_at, ''), COALESCE(promoted_to_id, '')
	          FROM mistakes_local WHERE deleted_at IS NULL`
	args := []any{}
	if tier != "" {
		query += ` AND tier = ?`
		args = append(args, tier)
	}
	query += ` ORDER BY last_hit_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query mistakes: %w", err)
	}
	defer rows.Close()

	var out []Mistake
	for rows.Next() {
		var m Mistake
		if err := rows.Scan(&m.ID, &m.Category, &m.Title, &m.Content,
			&m.ContextOrigin, &m.Tier, &m.HitCount,
			&m.LastHitAt, &m.CreatedAt, &m.PromotedAt, &m.PromotedToID); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) PruneMistakes(olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE mistakes_local SET deleted_at = CURRENT_TIMESTAMP, deleted_by = 'prune-cron'
		 WHERE deleted_at IS NULL AND tier = 'raw' AND last_hit_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("prune mistakes: %w", err)
	}
	return res.RowsAffected()
}

func (s *Store) CountMistakes(tier string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `SELECT COUNT(*) FROM mistakes_local WHERE deleted_at IS NULL`
	args := []any{}
	if tier != "" {
		query += ` AND tier = ?`
		args = append(args, tier)
	}

	var n int64
	if err := s.db.QueryRow(query, args...).Scan(&n); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}
