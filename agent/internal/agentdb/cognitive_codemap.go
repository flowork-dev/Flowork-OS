// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval (autonomy grant 2026-06-19).
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-20
// Reason: CGM codemap→graph bridge (LinkCodemapToGraph) — built + unit-tested, verified
//   live (346 nodes/599 edges mr-flow). Extend = new file (mis. semantic concept→code).
//
// cognitive_codemap.go — Phase 3C/D14/D15: jembatan CODEMAP → COGNITIVE GRAPH.
//
// File BARU (cognitive_graph.go locked — extend, jangan modify). Bikin Flowork
// gak cuma "hapal konteks" (twin) tapi "hapal lokasi tubuh": struktur kode dirinya
// sendiri jadi NODE di graph kesadaran, nyambung ke konsep/peristiwa.
//
// D14 (graph = overlay/indeks, bukan kopi): node code = gagang ringan —
//   id = URN "<scope>/codemap/<path>", source_ref = path (pointer balik ke
//   codemap_files), type = "code". Konten (summary/health/deps) TETAP di codemap.
//
// LINK STRUKTURAL (deterministik, NO-LLM → aman jalan kapan aja):
//   - codemap_file_edges (import A→B) → cognitive_edge depends_on.
//   - tiap file → part_of → node layer (code_layer:<layer>).
// LINK SEMANTIK (konsep/twin → kode) = digestion + embedding-resolution, NYUSUL
//   (butuh LLM). Itu yang bikin "di mana dream cycle gue?" → telusur graph ke file.

package agentdb

import "fmt"

// codemapFileRow — baris ringan dari codemap_files buat jadi node.
type codemapFileRow struct {
	path, name, layer string
	lineCount, health int
}

// LinkCodemapToGraph proyeksikan codemap (tubuh) ke cognitive graph (kesadaran).
// scope = prefix URN agent (mis. "agent:mr-flow"). Idempoten (UpsertNode/Edge).
// Return (nodesAdded, edgesAdded). NO-LLM, NO embedding (bisa di-backfill nanti).
func (s *Store) LinkCodemapToGraph(scope string) (int, int, error) {
	if scope == "" {
		scope = "agent:local"
	}
	// ── baca codemap (lock singkat, lepas sebelum Upsert biar ga deadlock) ──
	s.mu.Lock()
	s.ensureCognitiveGraphSchema()
	var files []codemapFileRow
	rows, err := s.db.Query(`SELECT path, name, layer, line_count, health_score FROM codemap_files`)
	if err != nil {
		s.mu.Unlock()
		return 0, 0, fmt.Errorf("read codemap_files: %w", err)
	}
	for rows.Next() {
		var f codemapFileRow
		if rows.Scan(&f.path, &f.name, &f.layer, &f.lineCount, &f.health) == nil {
			files = append(files, f)
		}
	}
	rows.Close()

	type fe struct{ from, to string }
	var fedges []fe
	er2, err := s.db.Query(`SELECT from_path, to_path FROM codemap_file_edges`)
	if err == nil {
		for er2.Next() {
			var e fe
			if er2.Scan(&e.from, &e.to) == nil {
				fedges = append(fedges, e)
			}
		}
		er2.Close()
	}
	s.mu.Unlock()

	if len(files) == 0 {
		return 0, 0, fmt.Errorf("codemap kosong — index dulu (codemap belum jalan?)")
	}

	codeID := func(path string) string { return scope + "/codemap/" + path }
	layerID := func(layer string) string { return scope + "/code_layer/" + slug(layer) }

	nodesAdded, edgesAdded := 0, 0
	layersSeen := map[string]bool{}

	// ── node per file + edge part_of → layer ──
	for _, f := range files {
		label := f.path // path = identitas tubuh yg paling berguna
		props := fmt.Sprintf(`{"line_count":%d,"health":%d,"layer":%q}`, f.lineCount, f.health, f.layer)
		added, _ := s.UpsertNode(CogNode{
			ID: codeID(f.path), Label: label, Type: "code",
			WhereDomain: "codebase", Properties: props,
			SourceKind: "verified", SourceRef: f.path, Confidence: 1.0, Status: "active",
		})
		if added {
			nodesAdded++
		}
		if f.layer != "" {
			lid := layerID(f.layer)
			if !layersSeen[f.layer] {
				layersSeen[f.layer] = true
				if a, _ := s.UpsertNode(CogNode{
					ID: lid, Label: f.layer, Type: "code_layer", WhereDomain: "codebase",
					SourceKind: "verified", SourceRef: "codemap:layer", Confidence: 1.0, Status: "active",
				}); a {
					nodesAdded++
				}
			}
			if s.UpsertEdge(CogEdge{
				FromID: codeID(f.path), ToID: lid, RelationType: "part_of",
				Confidence: 1.0, SourceKind: "verified", SourceRef: "codemap", Status: "active",
			}) == nil {
				edgesAdded++
			}
		}
	}

	// ── edge depends_on (import A→B) ──
	for _, e := range fedges {
		if e.from == "" || e.to == "" || e.from == e.to {
			continue
		}
		if s.UpsertEdge(CogEdge{
			FromID: codeID(e.from), ToID: codeID(e.to), RelationType: "depends_on",
			Confidence: 1.0, SourceKind: "verified", SourceRef: "codemap", Status: "active",
		}) == nil {
			edgesAdded++
		}
	}
	return nodesAdded, edgesAdded, nil
}
