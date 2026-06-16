// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30 · re-edit 2026-06-17 (owner-approved): Explore count LIVE drawers
//   (deleted_at IS NULL) — konsisten sama stats.go, Content overview gak nunjuk 5M tombstoned. Re-LOCK.
// Reason: Audit pass — Brain drawer/embeddings/skills.

// Brain content explorer (read-only).

package brain

import (
	"context"
	"database/sql"
)

// CountPair — a labelled count for breakdowns (categories, sources).
type CountPair struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// ExploreStats — content overview of the knowledge brain (read-only), mirroring
// the flowork "FQ-Brain Explorer" overview so the same insight lives in the router.
type ExploreStats struct {
	Available           bool        `json:"available"`
	DrawersActive       int64       `json:"drawersActive"`
	Constitution        int64       `json:"constitution"`
	Agents              int64       `json:"agents"`
	Memories            int64       `json:"memories"`
	Skills              int         `json:"skills"` // embedded skill library
	Categories          []CountPair `json:"categories"`
	ConstitutionSources []CountPair `json:"constitutionSources"`
}

// Explore returns a content overview. Each count is best-effort: a query error
// leaves that field at zero rather than failing the whole call.
func Explore(ctx context.Context) ExploreStats {
	st := ExploreStats{Skills: len(Skills())}
	if !Available() {
		return st
	}
	db, err := Open()
	if err != nil {
		return st
	}
	st.Available = true
	// LIVE drawers only (deleted_at IS NULL): per 2026-06-17 retrieval SKIP tombstoned,
	// jadi overview WAJIB konsisten sama stats.go (corpus retrievable beneran, bukan 5M
	// yg 83% soft-deleted). DrawersActive = ACTIVE = live.
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM drawers WHERE deleted_at IS NULL`).Scan(&st.DrawersActive)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM constitution WHERE deleted_at IS NULL`).Scan(&st.Constitution)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agents`).Scan(&st.Agents)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE deleted_at IS NULL`).Scan(&st.Memories)
	st.Categories = countPairs(ctx, db, `SELECT COALESCE(mem_type,'(none)'), COUNT(*) FROM drawers
		WHERE deleted_at IS NULL GROUP BY mem_type ORDER BY 2 DESC LIMIT 10`)
	st.ConstitutionSources = countPairs(ctx, db, `SELECT source_file, COUNT(*) FROM constitution
		WHERE deleted_at IS NULL GROUP BY source_file ORDER BY 2 DESC LIMIT 10`)
	return st
}

// ConstitutionEntry — one sacred rule.
type ConstitutionEntry struct {
	ID        int64   `json:"id"`
	Section   string  `json:"section"`
	Source    string  `json:"source"`
	Amplitude float64 `json:"amplitude"`
	Content   string  `json:"content"`
}

// ListConstitution returns sacred rules (highest amplitude first). limit<=0 → 100.
func ListConstitution(ctx context.Context, limit, maxContentLen int) ([]ConstitutionEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	db, err := Open()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, section, source_file, amplitude, content FROM constitution
		WHERE deleted_at IS NULL ORDER BY amplitude DESC, section ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ConstitutionEntry
	for rows.Next() {
		var e ConstitutionEntry
		if err := rows.Scan(&e.ID, &e.Section, &e.Source, &e.Amplitude, &e.Content); err != nil {
			continue
		}
		if maxContentLen > 0 {
			e.Content = truncateRunes(e.Content, maxContentLen)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// countPairs runs a "label, count" query and maps rows to CountPair.
func countPairs(ctx context.Context, db *sql.DB, query string) []CountPair {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []CountPair
	for rows.Next() {
		var p CountPair
		if err := rows.Scan(&p.Label, &p.Count); err == nil {
			out = append(out, p)
		}
	}
	return out
}
