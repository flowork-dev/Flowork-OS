// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30 · re-edit 2026-06-17 (owner-approved): JOIN drawers filter
//   deleted_at — drawer tombstoned (soft-delete) JANGAN di-retrieve (83% corpus). Re-LOCK.
// Reason: Audit pass — Brain drawer/embeddings/skills.

// Brain RAG retrieval (FTS5 BM25).

package brain

import (
	"context"
	"database/sql"
	"fmt"
)

// Snippet — one retrieved knowledge chunk from the Memory Palace.
type Snippet struct {
	DrawerID string  `json:"drawer_id"`
	Wing     string  `json:"wing"`
	Room     string  `json:"room"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"` // normalized relevance, higher = better, range (0,1]
}

// RetrieveOpts controls a retrieval.
type RetrieveOpts struct {
	// Limit — max snippets to return. <=0 defaults to 6.
	Limit int
	// Wings — restrict to these wings. Empty = search all wings.
	Wings []string
	// MaxContentLen — truncate each snippet's content to this many runes (0 = no cap).
	MaxContentLen int
}

// Retrieve runs an FTS5 BM25 search over the Memory Palace and returns the
// top-N most relevant drawers for RAG context injection. This is the L2 layer
// of the cascade, exposed as multi-result for prompt enrichment.
// Performance: content/wing/room live in memory_fts (no JOIN needed for THOSE);
// the only JOIN is a cheap PK existence-check on drawers.deleted_at (skip
// tombstoned — 2026-06-17), applied AFTER the MATCH narrows to a small set.
// AND-first matching on a 30GB+ DB is ~10x faster than OR while returning the
// same top hits. If the stricter AND match yields nothing, falls back to OR.
// BM25 returns lower scores for more relevant rows, so we invert to a positive
// (0,1] relevance.
func Retrieve(ctx context.Context, db *sql.DB, query string, opts RetrieveOpts) ([]Snippet, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 6
	}
	tokens := ftsTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	// Precision-first: AND match (small set, fast). Fall back to OR only if AND
	// finds nothing — that fallback set is small too (rare terms).
	snips, err := runFTS(ctx, db, joinFTS(tokens, "AND"), opts.Wings, limit, opts.MaxContentLen)
	if err != nil {
		return nil, err
	}
	if len(snips) == 0 && len(tokens) > 1 {
		snips, err = runFTS(ctx, db, joinFTS(tokens, "OR"), opts.Wings, limit, opts.MaxContentLen)
		if err != nil {
			return nil, err
		}
	}
	return snips, nil
}

// runFTS executes one FTS5 MATCH and maps rows to Snippets.
func runFTS(ctx context.Context, db *sql.DB, match string, wings []string, limit, maxLen int) ([]Snippet, error) {
	var (
		rows *sql.Rows
		err  error
	)
	// JOIN drawers buat filter deleted_at: drawer SOFT-DELETED (tombstone) JANGAN
	// di-retrieve (owner 2026-06-17 — 83% corpus tombstoned bikin brain nyajiin data
	// ke-hapus). JOIN = PK-lookup atas hasil MATCH (set kecil, AND-first) → murah,
	// BUKAN scan 5M. FTS5 MATCH wajib nama-tabel PENUH (alias bikin MATCH gagal).
	if len(wings) > 0 {
		placeholders := ""
		args := []any{match}
		for i, w := range wings {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, w)
		}
		args = append(args, limit)
		stmt := fmt.Sprintf(`SELECT %[1]s.drawer_id, %[1]s.wing, %[1]s.room, %[1]s.content, bm25(%[1]s) AS score
			FROM %[1]s JOIN drawers d ON d.id = %[1]s.drawer_id
			WHERE %[1]s MATCH ? AND d.deleted_at IS NULL AND %[1]s.wing IN (%[2]s)
			ORDER BY score LIMIT ?`, ftsTable, placeholders)
		rows, err = db.QueryContext(ctx, stmt, args...)
	} else {
		stmt := fmt.Sprintf(`SELECT %[1]s.drawer_id, %[1]s.wing, %[1]s.room, %[1]s.content, bm25(%[1]s) AS score
			FROM %[1]s JOIN drawers d ON d.id = %[1]s.drawer_id
			WHERE %[1]s MATCH ? AND d.deleted_at IS NULL
			ORDER BY score LIMIT ?`, ftsTable)
		rows, err = db.QueryContext(ctx, stmt, match, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("brain retrieve: %w", err)
	}
	defer rows.Close()

	var out []Snippet
	for rows.Next() {
		var s Snippet
		var bm25 float64
		if err := rows.Scan(&s.DrawerID, &s.Wing, &s.Room, &s.Content, &bm25); err != nil {
			continue
		}
		if bm25 < 0 {
			bm25 = -bm25
		}
		s.Score = 1.0 / (1.0 + bm25)
		if maxLen > 0 {
			s.Content = truncateRunes(s.Content, maxLen)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("brain retrieve rows: %w", err)
	}
	return out, nil
}

// truncateRunes caps a string to n runes, appending an ellipsis when cut.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
