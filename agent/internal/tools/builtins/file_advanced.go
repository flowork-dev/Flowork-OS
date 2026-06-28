// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"flowork-gui/internal/tools"
)

type editTool struct{}

func (editTool) Name() string       { return "edit" }
func (editTool) Capability() string { return "fs:write:/shared/*" }
func (editTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Replace an exact substring in a file. Preferred: file_path (relative path in your workspace). Legacy: category + name. Default replace_all=false → reject if >1 match. File cap 4MB.",
		Params: []tools.Param{
			{Name: "file_path", Type: tools.ParamString, Description: "relative path in your workspace (preferred). Absolute/'..' rejected (isolation)."},
			{Name: "old_string", Type: tools.ParamString, Description: "exact substring to find", Required: true},
			{Name: "new_string", Type: tools.ParamString, Description: "replacement", Required: true},
			{Name: "replace_all", Type: tools.ParamBool, Description: "default false; true = replace all occurrences"},
			{Name: "category", Type: tools.ParamString, Description: "legacy: tools|job|document|media|cache|log (use file_path instead)"},
			{Name: "name", Type: tools.ParamString, Description: "legacy: filename (basename only) — pair with category"},
		},
		Returns: "{replaced: <count>, length}",
	}
}
func (editTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	oldS, _ := args["old_string"].(string)
	newS, _ := args["new_string"].(string)
	replaceAll, _ := args["replace_all"].(bool)

	if oldS == "" {
		return tools.Result{}, fmt.Errorf("old_string required (non-empty)")
	}

	abs, _, err := resolveFileArgs(ctx, args)
	if err != nil {
		return tools.Result{}, err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return tools.Result{}, fmt.Errorf("read %s: %w", filepath.Base(abs), err)
	}
	if len(data) > maxFileBytes {
		return tools.Result{}, fmt.Errorf("file too large (>4MB)")
	}
	content := string(data)
	count := strings.Count(content, oldS)
	if count == 0 {
		return tools.Result{}, fmt.Errorf("old_string not found")
	}
	if count > 1 && !replaceAll {
		return tools.Result{}, fmt.Errorf("old_string matches %d places; set replace_all=true or narrow old_string", count)
	}
	var updated string
	if replaceAll {
		updated = strings.ReplaceAll(content, oldS, newS)
	} else {
		updated = strings.Replace(content, oldS, newS, 1)
	}
	if len(updated) > maxFileBytes {
		return tools.Result{}, fmt.Errorf("result exceeds 4MB cap")
	}
	if err := os.WriteFile(abs, []byte(updated), 0o644); err != nil {
		return tools.Result{}, fmt.Errorf("write %s: %w", filepath.Base(abs), err)
	}
	return tools.Result{Output: map[string]any{
		"replaced": count,
		"length":   len(updated),
	}}, nil
}

type globTool struct{}

func (globTool) Name() string       { return "glob" }
func (globTool) Capability() string { return "fs:read:/shared/*" }
func (globTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "List files matching glob pattern (mis. '*.md', 'document/*.txt'). Recursive scan disabled — top-level + category subdirs only. Cap 200 results. Symlinks skipped.",
		Params: []tools.Param{
			{Name: "pattern", Type: tools.ParamString, Description: "glob pattern relative ke shared workspace", Required: true},
		},
		Returns: "{files: [...], count, truncated}",
	}
}
func (globTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	pattern, _ := args["pattern"].(string)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return tools.Result{}, fmt.Errorf("pattern required")
	}
	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return tools.Result{}, fmt.Errorf("shared workspace not in context")
	}

	if filepath.IsAbs(pattern) || strings.Contains(pattern, "..") {
		return tools.Result{}, fmt.Errorf("pattern must be relative + no '..'")
	}

	const cap = 200
	results := []string{}
	truncated := false

	for category := range fileCategoryWhitelist {
		dir := filepath.Join(shared, category)

		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, m := range matches {

			fi, err := os.Lstat(m)
			if err != nil || fi.Mode()&os.ModeSymlink != 0 || fi.IsDir() {
				continue
			}
			rel, _ := filepath.Rel(shared, m)
			results = append(results, rel)
			if len(results) >= cap {
				truncated = true
				break
			}
		}
		if truncated {
			break
		}
	}

	if !truncated {
		extra, _ := filepath.Glob(filepath.Join(shared, pattern))
		for _, m := range extra {
			fi, err := os.Lstat(m)
			if err != nil || fi.Mode()&os.ModeSymlink != 0 || fi.IsDir() {
				continue
			}
			rel, _ := filepath.Rel(shared, m)

			seen := false
			for _, r := range results {
				if r == rel {
					seen = true
					break
				}
			}
			if !seen {
				results = append(results, rel)
				if len(results) >= cap {
					truncated = true
					break
				}
			}
		}
	}

	sort.Strings(results)
	return tools.Result{Output: map[string]any{
		"files":     results,
		"count":     len(results),
		"truncated": truncated,
	}}, nil
}

type grepTool struct{}

func (grepTool) Name() string       { return "grep" }
func (grepTool) Capability() string { return "fs:read:/shared/*" }
func (grepTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Search lines matching pattern across shared workspace. Default substring (case-sensitive). regex=true → Go regexp. category filter optional. Cap 200 hits + 4MB total scanned.",
		Params: []tools.Param{
			{Name: "pattern", Type: tools.ParamString, Description: "search pattern", Required: true},
			{Name: "category", Type: tools.ParamString, Description: "optional filter to one category"},
			{Name: "regex", Type: tools.ParamBool, Description: "default false; true = treat pattern as Go regexp"},
		},
		Returns: "{hits: [{file, line_no, line}], count, truncated, scanned_bytes}",
	}
}
func (grepTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	pattern, _ := args["pattern"].(string)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return tools.Result{}, fmt.Errorf("pattern required")
	}
	category, _ := args["category"].(string)
	useRegex, _ := args["regex"].(bool)

	shared := tools.FromSharedDir(ctx)
	if shared == "" {
		return tools.Result{}, fmt.Errorf("shared workspace not in context")
	}

	var re *regexp.Regexp
	if useRegex {
		r, rerr := regexp.Compile(pattern)
		if rerr != nil {
			return tools.Result{}, fmt.Errorf("invalid regex: %w", rerr)
		}
		re = r
	}

	const (
		hitCap   = 200
		scanCap  = 4 * 1024 * 1024
		lineSize = 4 * 1024
	)
	type hit struct {
		File   string `json:"file"`
		LineNo int    `json:"line_no"`
		Line   string `json:"line"`
	}
	hits := []hit{}
	scanned := 0
	truncated := false

	categories := []string{}
	if category != "" {
		if _, ok := fileCategoryWhitelist[category]; !ok {
			return tools.Result{}, fmt.Errorf("category %q not in whitelist", category)
		}
		categories = []string{category}
	} else {
		for c := range fileCategoryWhitelist {
			categories = append(categories, c)
		}
	}

	for _, c := range categories {
		dir := filepath.Join(shared, c)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			abs := filepath.Join(dir, e.Name())
			fi, err := os.Lstat(abs)
			if err != nil || fi.Mode()&os.ModeSymlink != 0 {
				continue
			}
			f, oerr := os.Open(abs)
			if oerr != nil {
				continue
			}
			scanner := bufio.NewScanner(f)
			scanner.Buffer(make([]byte, lineSize), lineSize)
			lineNo := 0
			for scanner.Scan() {
				lineNo++
				line := scanner.Text()
				scanned += len(line) + 1
				if scanned > scanCap {
					truncated = true
					f.Close()
					goto DONE
				}
				match := false
				if useRegex {
					match = re.MatchString(line)
				} else {
					match = strings.Contains(line, pattern)
				}
				if !match {
					continue
				}
				if len(line) > 240 {
					line = line[:240] + "…"
				}
				hits = append(hits, hit{
					File:   filepath.Join(c, e.Name()),
					LineNo: lineNo,
					Line:   line,
				})
				if len(hits) >= hitCap {
					truncated = true
					f.Close()
					goto DONE
				}
			}

			if serr := scanner.Err(); serr != nil {
				truncated = true
			}
			f.Close()
		}
	}
DONE:

	out := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		out = append(out, map[string]any{
			"file":    h.File,
			"line_no": h.LineNo,
			"line":    h.Line,
		})
	}
	return tools.Result{Output: map[string]any{
		"hits":          out,
		"count":         len(hits),
		"truncated":     truncated,
		"scanned_bytes": scanned,
	}}, nil
}

// SELF-REGISTER plug-in (📄 lock/tool-manager.md) — edit/hapus/tambah BEBAS tanpa unfreeze.
func init() {
	tools.Register(&editTool{})
	tools.Register(&globTool{})
	tools.Register(&grepTool{})
}
