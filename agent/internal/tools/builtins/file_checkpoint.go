// file_checkpoint.go — F-G #1: jaring pengaman evolusi (checkpoint otomatis + undo).
// 📄 Dok: FLowork_os/lock/file-checkpoint.md
//
// NON-FROZEN sibling (deletable): dicolok via tools.RegisterInterceptor +
// tools.Register — core frozen NOL disentuh; file ini dihapus → fiturnya ilang,
// rumah tetep berdiri (prinsip switch).
//
// Sebelum file_write/edit nyentuh disk, isi LAMA file di-snapshot ke
// <workspace>/.checkpoints/ (cap 100 per-agent — tertua dibuang). File yang
// BELUM ada dapet marker "absent" → undo pembuatan = hapus file. Gagal snapshot
// TIDAK pernah ngeblok tulisan (best-effort: ini safety net, bukan gerbang) —
// kode agent salah/typo/error tetep bisa DIBALIKIN, ga ngerusak yang udah ada.
//
// Tool:
//   file_checkpoints (read-only) — list snapshot sebuah file (id + waktu + ukuran)
//   undo_file                    — balikin file ke snapshot (default terbaru);
//                                  kondisi sekarang di-snapshot dulu → redo bisa.
package builtins

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"flowork-gui/internal/tools"
)

const (
	ckptDirName   = ".checkpoints"
	ckptKeep      = 100
	ckptSuffix    = ".snap"
	ckptAbsentSuf = ".snap.absent" // snapshot "file belum ada" (undo create = hapus)
)

func init() {
	tools.RegisterInterceptor(fileCheckpointInterceptor{})
	tools.Register(&undoFileTool{})
	tools.Register(&fileCheckpointsTool{})
}

// ── interceptor: snapshot sebelum mutasi ───────────────────────────────────

type fileCheckpointInterceptor struct{}

func (fileCheckpointInterceptor) Name() string { return "file-checkpoint" }

func (fileCheckpointInterceptor) Before(ctx context.Context, t tools.Tool, args map[string]any) error {
	switch t.Name() {
	case "file_write", "edit":
	default:
		return nil
	}
	abs, rel, err := resolveFileArgs(ctx, args)
	if err != nil {
		return nil // resolusi gagal → biar tool-nya sendiri yang ngelapor
	}
	if serr := ckptSnapshot(ctx, abs, rel); serr != nil {
		// BEST-EFFORT: jangan pernah blok tulisan gara-gara snapshot gagal.
		fmt.Fprintf(os.Stderr, "[file-checkpoint] snapshot %s gagal (non-blocking): %v\n", rel, serr)
	}
	return nil
}

// ── mesin snapshot ──────────────────────────────────────────────────────────

func ckptRoot(ctx context.Context) (string, error) {
	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return "", fmt.Errorf("shared workspace not in context")
	}
	return filepath.Join(shared, ckptDirName), nil
}

func ckptEncodeRel(rel string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(rel))
}

// ckptSnapshot — simpan isi abs SEKARANG sebagai snapshot untuk rel.
func ckptSnapshot(ctx context.Context, abs, rel string) error {
	root, err := ckptRoot(ctx)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	base := fmt.Sprintf("%019d__%s", time.Now().UnixNano(), ckptEncodeRel(rel))
	data, rerr := os.ReadFile(abs)
	switch {
	case rerr == nil:
		if len(data) > maxFileBytes {
			return fmt.Errorf("file >4MB, skip snapshot")
		}
		if err := os.WriteFile(filepath.Join(root, base+ckptSuffix), data, 0o644); err != nil {
			return err
		}
	case os.IsNotExist(rerr):
		if err := os.WriteFile(filepath.Join(root, base+ckptAbsentSuf), nil, 0o644); err != nil {
			return err
		}
	default:
		return rerr
	}
	ckptPrune(root)
	return nil
}

// ckptPrune — jaga max ckptKeep snapshot (semua file, semua target); tertua dibuang.
func ckptPrune(root string) {
	ents, err := os.ReadDir(root)
	if err != nil || len(ents) <= ckptKeep {
		return
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() && strings.Contains(e.Name(), "__") {
			names = append(names, e.Name())
		}
	}
	if len(names) <= ckptKeep {
		return
	}
	sort.Strings(names) // prefix UnixNano 19-digit zero-pad → urutan waktu
	for _, n := range names[:len(names)-ckptKeep] {
		_ = os.Remove(filepath.Join(root, n))
	}
}

type ckptEntry struct {
	ID     int64
	Absent bool
	Path   string
	Bytes  int64
}

// ckptList — snapshot milik rel, TERBARU duluan.
func ckptList(root, rel string) []ckptEntry {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	marker := "__" + ckptEncodeRel(rel)
	var out []ckptEntry
	for _, e := range ents {
		name := e.Name()
		absent := strings.HasSuffix(name, ckptAbsentSuf)
		if !absent && !strings.HasSuffix(name, ckptSuffix) {
			continue
		}
		stem := strings.TrimSuffix(strings.TrimSuffix(name, ckptAbsentSuf), ckptSuffix)
		if !strings.HasSuffix(stem, marker) {
			continue
		}
		ts, perr := strconv.ParseInt(strings.SplitN(stem, "__", 2)[0], 10, 64)
		if perr != nil {
			continue
		}
		var sz int64
		if fi, ferr := e.Info(); ferr == nil {
			sz = fi.Size()
		}
		out = append(out, ckptEntry{ID: ts, Absent: absent, Path: filepath.Join(root, name), Bytes: sz})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

// ── tool: file_checkpoints (read-only) ──────────────────────────────────────

type fileCheckpointsTool struct{}

func (fileCheckpointsTool) Name() string       { return "file_checkpoints" }
func (fileCheckpointsTool) Capability() string { return "fs:read:/shared/*" }
func (fileCheckpointsTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "List snapshot otomatis sebuah file (dibikin tiap file_write/edit, max 100). Pakai sebelum undo_file buat milih checkpoint.",
		Params: []tools.Param{
			{Name: "file_path", Type: tools.ParamString, Description: "relative path in your workspace (preferred)."},
			{Name: "category", Type: tools.ParamString, Description: "legacy: tools|job|document|media|cache|log"},
			{Name: "name", Type: tools.ParamString, Description: "legacy: filename — pair with category"},
		},
		Returns: "{checkpoints:[{id, time, bytes, was_absent}], count}",
	}
}
func (fileCheckpointsTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	_, rel, err := resolveFileArgs(ctx, args)
	if err != nil {
		return tools.Result{}, err
	}
	root, err := ckptRoot(ctx)
	if err != nil {
		return tools.Result{}, err
	}
	list := ckptList(root, rel)
	items := make([]map[string]any, 0, len(list))
	for _, c := range list {
		items = append(items, map[string]any{
			"id":         strconv.FormatInt(c.ID, 10),
			"time":       time.Unix(0, c.ID).Format(time.RFC3339),
			"bytes":      c.Bytes,
			"was_absent": c.Absent,
		})
	}
	return tools.Result{Output: map[string]any{"checkpoints": items, "count": len(items)}}, nil
}

// ── tool: undo_file ─────────────────────────────────────────────────────────

type undoFileTool struct{}

func (undoFileTool) Name() string       { return "undo_file" }
func (undoFileTool) Capability() string { return "fs:write:/shared/*" }
func (undoFileTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Balikin file ke snapshot checkpoint (default: TERBARU). Kondisi sekarang di-snapshot dulu → undo bisa di-undo (redo). Kalau checkpoint-nya penanda 'belum ada', file DIHAPUS (undo pembuatan).",
		Params: []tools.Param{
			{Name: "file_path", Type: tools.ParamString, Description: "relative path in your workspace (preferred)."},
			{Name: "checkpoint", Type: tools.ParamString, Description: "optional: id dari file_checkpoints (default snapshot terbaru)"},
			{Name: "category", Type: tools.ParamString, Description: "legacy: tools|job|document|media|cache|log"},
			{Name: "name", Type: tools.ParamString, Description: "legacy: filename — pair with category"},
		},
		Returns: "{path, restored_checkpoint, bytes, deleted}",
	}
}
func (undoFileTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	abs, rel, err := resolveFileArgs(ctx, args)
	if err != nil {
		return tools.Result{}, err
	}
	root, err := ckptRoot(ctx)
	if err != nil {
		return tools.Result{}, err
	}
	list := ckptList(root, rel)
	if len(list) == 0 {
		return tools.Result{}, fmt.Errorf("belum ada checkpoint buat %q — snapshot kebentuk otomatis tiap file_write/edit", rel)
	}
	pick := list[0]
	if want := strings.TrimSpace(fmt.Sprint(args["checkpoint"])); want != "" && want != "<nil>" {
		found := false
		for _, c := range list {
			if strconv.FormatInt(c.ID, 10) == want {
				pick, found = c, true
				break
			}
		}
		if !found {
			return tools.Result{}, fmt.Errorf("checkpoint %s ga ketemu buat %q (cek file_checkpoints)", want, rel)
		}
	}
	// Simpan kondisi SEKARANG dulu → redo bisa (undo-nya undo).
	if serr := ckptSnapshot(ctx, abs, rel); serr != nil {
		fmt.Fprintf(os.Stderr, "[file-checkpoint] pre-undo snapshot %s gagal (lanjut): %v\n", rel, serr)
	}
	if pick.Absent {
		if rerr := os.Remove(abs); rerr != nil && !os.IsNotExist(rerr) {
			return tools.Result{}, fmt.Errorf("hapus (undo create): %w", rerr)
		}
		return tools.Result{Output: map[string]any{
			"path": rel, "restored_checkpoint": strconv.FormatInt(pick.ID, 10),
			"bytes": 0, "deleted": true,
		}}, nil
	}
	data, rerr := os.ReadFile(pick.Path)
	if rerr != nil {
		return tools.Result{}, fmt.Errorf("baca snapshot: %w", rerr)
	}
	if merr := os.MkdirAll(filepath.Dir(abs), 0o755); merr != nil {
		return tools.Result{}, fmt.Errorf("mkdir: %w", merr)
	}
	if werr := os.WriteFile(abs, data, 0o644); werr != nil {
		return tools.Result{}, fmt.Errorf("restore: %w", werr)
	}
	return tools.Result{Output: map[string]any{
		"path": rel, "restored_checkpoint": strconv.FormatInt(pick.ID, 10),
		"bytes": len(data), "deleted": false,
	}}, nil
}
