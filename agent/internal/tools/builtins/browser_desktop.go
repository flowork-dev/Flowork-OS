//go:build (linux || darwin || windows) && !android

// browser_desktop.go — browser-control NATIVE (Go, no node), Opsi B roadmap multi-os-tools.
// Drive Chromium yg udah ada (image flowork-os bundle chromium) lewat CDP via go-rod —
// TANPA node/chrome-devtools-mcp. Pendekatan dicontek dari chrome-devtools-mcp (Antigravity):
// navigate + a11y/interactive snapshot (aksi by-uid) + click/type/fill/upload + screenshot +
// SET COOKIES (inject sesi login). Build-tag DESKTOP ONLY (android set tag `linux` juga →
// wajib `&& !android`; android pakai Accessibility, track terpisah).
//
// CAPABILITY: browser:control. Lifecycle: chromium di-launch lazy (headless, profil persisten
// → cookie nempel) ATAU connect ke Chrome jalan (FLOWORK_BROWSER_URL). Binari: FLOWORK_CHROME_BIN
// atau auto-detect chromium/google-chrome.

package builtins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"flowork-gui/internal/tools"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func init() {
	tools.Register(&browserNavigateTool{})
	tools.Register(&browserSnapshotTool{})
	tools.Register(&browserClickTool{})
	tools.Register(&browserTypeTool{})
	tools.Register(&browserUploadTool{})
	tools.Register(&browserScreenshotTool{})
	tools.Register(&browserSetCookiesTool{})
	tools.Register(&browserEvalTool{})
	tools.Register(&browserCloseTool{})
}

const browserCap = "browser:control"

// ── lifecycle (singleton browser + page) ────────────────────────────────────
var (
	brMu       sync.Mutex
	brInst     *rod.Browser
	brPage     *rod.Page
	brLastUsed time.Time
	brReaper   sync.Once
)

func chromeBin() string {
	if b := strings.TrimSpace(os.Getenv("FLOWORK_CHROME_BIN")); b != "" {
		return b
	}
	for _, c := range []string{
		"/usr/bin/chromium", "/usr/bin/chromium-browser",
		"/usr/bin/google-chrome-stable", "/usr/bin/google-chrome",
		"/snap/bin/chromium",
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "" // go-rod akan coba auto (ga ideal di standalone)
}

func getPage() (*rod.Page, error) {
	brMu.Lock()
	defer brMu.Unlock()
	brLastUsed = time.Now()
	brReaper.Do(startBrowserReaper) // jaring pengaman: auto-close browser nganggur >30mnt
	if brInst != nil && brPage != nil {
		return brPage, nil
	}
	// connect-mode: nempel ke Chrome jalan (sesi+cookie login kepakai).
	if url := strings.TrimSpace(os.Getenv("FLOWORK_BROWSER_URL")); url != "" {
		b := rod.New().ControlURL(url)
		if err := b.Connect(); err != nil {
			return nil, fmt.Errorf("connect FLOWORK_BROWSER_URL: %w", err)
		}
		p, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			return nil, fmt.Errorf("new page: %w", err)
		}
		brInst, brPage = b, p
		return p, nil
	}
	// launch-mode: chromium headless + profil persisten (cookie nempel antar-sesi).
	profile := filepath.Join(os.TempDir(), "flowork-browser-profile")
	if home, herr := os.UserHomeDir(); herr == nil {
		profile = filepath.Join(home, ".flowork", "browser-profile")
	}
	_ = os.MkdirAll(profile, 0o755)
	l := launcher.New().
		Headless(true).
		Set("no-sandbox").       // OS image / root container butuh ini
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		UserDataDir(profile)
	if bin := chromeBin(); bin != "" {
		l = l.Bin(bin)
	}
	ctrlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch chromium: %w", err)
	}
	b := rod.New().ControlURL(ctrlURL)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("connect chromium: %w", err)
	}
	p, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}
	brInst, brPage = b, p
	return p, nil
}

// pageTimeout — wrap page dgn timeout biar tool ga gantung selamanya.
func pageTimeout(p *rod.Page, d time.Duration) *rod.Page { return p.Timeout(d) }

// ── 1. browser_navigate ─────────────────────────────────────────────────────
type browserNavigateTool struct{}

func (browserNavigateTool) Name() string       { return "browser_navigate" }
func (browserNavigateTool) Capability() string { return browserCap }
func (browserNavigateTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Buka URL di browser asli (Chromium, lewat sesi login kalau di-set). Lolos JS/Cloudflare yg webfetch ga bisa. Setelah ini pakai browser_snapshot buat 'lihat' halaman.",
		Params: []tools.Param{
			{Name: "url", Type: tools.ParamString, Description: "absolute http(s) URL", Required: true},
		},
		Returns: "{url, title}",
	}
}
func (browserNavigateTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	url, _ := args["url"].(string)
	if strings.TrimSpace(url) == "" {
		return tools.Result{}, fmt.Errorf("url required")
	}
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	p = pageTimeout(p, 45*time.Second)
	if err := p.Navigate(url); err != nil {
		return tools.Result{}, fmt.Errorf("navigate: %w", err)
	}
	if err := p.WaitLoad(); err != nil {
		return tools.Result{}, fmt.Errorf("wait load: %w", err)
	}
	info, _ := p.Info()
	title := ""
	if info != nil {
		title = info.Title
	}
	return tools.Result{Output: map[string]any{"url": url, "title": title}}, nil
}

// ── 2. browser_snapshot (a11y/interactive — aksi by-uid) ────────────────────
const snapshotJS = `() => {
  const out = [];
  let n = 0;
  const sel = 'a,button,input,textarea,select,[role=button],[role=link],[role=textbox],[contenteditable=true]';
  document.querySelectorAll(sel).forEach(el => {
    const r = el.getBoundingClientRect();
    if (r.width === 0 && r.height === 0) return;          // skip hidden
    const style = getComputedStyle(el);
    if (style.visibility === 'hidden' || style.display === 'none') return;
    n++;
    el.setAttribute('data-fw-uid', String(n));
    const label = (el.getAttribute('aria-label') || el.placeholder || el.value || el.innerText || el.getAttribute('name') || '').trim().slice(0,80);
    out.push({uid:n, tag:el.tagName.toLowerCase(), type:(el.type||''), role:(el.getAttribute('role')||''), label:label, href:(el.href||'')});
  });
  return JSON.stringify({title:document.title, url:location.href, count:n, elements:out});
}`

type browserSnapshotTool struct{}

func (browserSnapshotTool) Name() string       { return "browser_snapshot" }
func (browserSnapshotTool) Capability() string { return browserCap }
func (browserSnapshotTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Snapshot elemen interaktif halaman saat ini (link/tombol/input) + uid masing2. Pakai uid buat browser_click/browser_type/browser_upload. Lebih akurat & hemat dari screenshot — UTAMAKAN ini.",
		Params:      nil,
		Returns:     "{title, url, count, elements:[{uid, tag, type, role, label, href}]}",
	}
}
func (browserSnapshotTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	p = pageTimeout(p, 20*time.Second)
	res, err := p.Eval(snapshotJS)
	if err != nil {
		return tools.Result{}, fmt.Errorf("snapshot eval: %w", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(res.Value.Str()), &parsed); err != nil {
		return tools.Result{}, fmt.Errorf("snapshot parse: %w", err)
	}
	return tools.Result{Output: parsed}, nil
}

// elByUID — cari elemen yg di-tag snapshot terakhir.
func elByUID(p *rod.Page, uid string) (*rod.Element, error) {
	if strings.TrimSpace(uid) == "" {
		return nil, fmt.Errorf("uid required (jalanin browser_snapshot dulu)")
	}
	el, err := p.Element(`[data-fw-uid="` + uid + `"]`)
	if err != nil {
		return nil, fmt.Errorf("uid %s ga ketemu — snapshot ulang (halaman mungkin berubah): %w", uid, err)
	}
	return el, nil
}

// ── 3. browser_click ────────────────────────────────────────────────────────
type browserClickTool struct{}

func (browserClickTool) Name() string       { return "browser_click" }
func (browserClickTool) Capability() string { return browserCap }
func (browserClickTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Klik elemen by uid (dari browser_snapshot).",
		Params:      []tools.Param{{Name: "uid", Type: tools.ParamString, Description: "uid elemen", Required: true}},
		Returns:     "{clicked: uid}",
	}
}
func (browserClickTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	uid, _ := args["uid"].(string)
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	p = pageTimeout(p, 20*time.Second)
	el, err := elByUID(p, uid)
	if err != nil {
		return tools.Result{}, err
	}
	// go-rod el.Click nge-wait stable/interactable → HANG di SPA berat (FB "context
	// deadline"). Pakai native JS .click() (no-wait); React/Lexical nangkep via event
	// delegation. Cabut-akar fix (2026-06-23).
	if _, err := el.Eval("() => this.click()"); err != nil {
		return tools.Result{}, fmt.Errorf("click: %w", err)
	}
	return tools.Result{Output: map[string]any{"clicked": uid}}, nil
}

// ── 4. browser_type (isi input by uid) ──────────────────────────────────────
type browserTypeTool struct{}

func (browserTypeTool) Name() string       { return "browser_type" }
func (browserTypeTool) Capability() string { return browserCap }
func (browserTypeTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Ketik teks ke input/textarea by uid (dari browser_snapshot). Buat isi form/login/search.",
		Params: []tools.Param{
			{Name: "uid", Type: tools.ParamString, Description: "uid input", Required: true},
			{Name: "text", Type: tools.ParamString, Description: "teks yg diketik", Required: true},
		},
		Returns: "{typed: uid}",
	}
}
func (browserTypeTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	uid, _ := args["uid"].(string)
	text, _ := args["text"].(string)
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	p = pageTimeout(p, 20*time.Second)
	el, err := elByUID(p, uid)
	if err != nil {
		return tools.Result{}, err
	}
	// focus + select-all (kalau input) via JS, lalu CDP InsertText (no-wait, dispatch
	// input event yg React/Lexical nangkep). el.Input go-rod bisa HANG di SPA berat.
	_, _ = el.Eval("() => { this.focus && this.focus(); if (this.select) this.select(); }")
	if err := p.InsertText(text); err != nil {
		return tools.Result{}, fmt.Errorf("type: %w", err)
	}
	return tools.Result{Output: map[string]any{"typed": uid}}, nil
}

// ── 5. browser_upload (upload file by uid — kasus upload video) ──────────────
type browserUploadTool struct{}

func (browserUploadTool) Name() string       { return "browser_upload" }
func (browserUploadTool) Capability() string { return browserCap }
func (browserUploadTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Upload file lokal ke input file by uid (dari browser_snapshot). Buat upload video/gambar/dokumen ke web.",
		Params: []tools.Param{
			{Name: "uid", Type: tools.ParamString, Description: "uid input file", Required: true},
			{Name: "path", Type: tools.ParamString, Description: "path file lokal absolut", Required: true},
		},
		Returns: "{uploaded: path}",
	}
}
func (browserUploadTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	uid, _ := args["uid"].(string)
	path, _ := args["path"].(string)
	if _, err := os.Stat(path); err != nil {
		return tools.Result{}, fmt.Errorf("file ga ada: %s", path)
	}
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	p = pageTimeout(p, 30*time.Second)
	el, err := elByUID(p, uid)
	if err != nil {
		return tools.Result{}, err
	}
	if err := el.SetFiles([]string{path}); err != nil {
		return tools.Result{}, fmt.Errorf("set files: %w", err)
	}
	return tools.Result{Output: map[string]any{"uploaded": path}}, nil
}

// ── 6. browser_screenshot (visual — simpan ke file) ─────────────────────────
type browserScreenshotTool struct{}

func (browserScreenshotTool) Name() string       { return "browser_screenshot" }
func (browserScreenshotTool) Capability() string { return browserCap }
func (browserScreenshotTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Screenshot halaman saat ini → simpan PNG ke file, balikin path. Buat kasus visual (layout/captcha). Untuk aksi, UTAMAKAN browser_snapshot.",
		Params:      []tools.Param{{Name: "path", Type: tools.ParamString, Description: "path simpan PNG (opsional)", Required: false}},
		Returns:     "{path, bytes}",
	}
}
func (browserScreenshotTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	path, _ := args["path"].(string)
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(os.TempDir(), "flowork-shot.png")
	}
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	p = pageTimeout(p, 20*time.Second)
	img, err := p.Screenshot(false, nil)
	if err != nil {
		return tools.Result{}, fmt.Errorf("screenshot: %w", err)
	}
	if err := os.WriteFile(path, img, 0o644); err != nil {
		return tools.Result{}, fmt.Errorf("write png: %w", err)
	}
	return tools.Result{Output: map[string]any{"path": path, "bytes": len(img)}}, nil
}

// ── 7. browser_set_cookies (INJECT sesi login — syarat owner) ───────────────
type browserSetCookiesTool struct{}

func (browserSetCookiesTool) Name() string       { return "browser_set_cookies" }
func (browserSetCookiesTool) Capability() string { return browserCap }
func (browserSetCookiesTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Inject cookies ke browser (sesi login tanpa ketik password). cookies = JSON array [{name,value,domain,path?}]. Setelah inject, navigate ke situsnya → udah login.",
		Params: []tools.Param{
			{Name: "cookies", Type: tools.ParamString, Description: `JSON array cookie: [{"name":"c_user","value":"...","domain":".facebook.com","path":"/"}]`, Required: true},
		},
		Returns: "{set: count}",
	}
}
func (browserSetCookiesTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	raw, _ := args["cookies"].(string)
	var items []struct {
		Name, Value, Domain, Path string
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return tools.Result{}, fmt.Errorf("cookies parse (harus JSON array {name,value,domain,path}): %w", err)
	}
	if len(items) == 0 {
		return tools.Result{}, fmt.Errorf("cookies kosong")
	}
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	cookies := make([]*proto.NetworkCookieParam, 0, len(items))
	for _, it := range items {
		path := it.Path
		if path == "" {
			path = "/"
		}
		cookies = append(cookies, &proto.NetworkCookieParam{
			Name: it.Name, Value: it.Value, Domain: it.Domain, Path: path,
		})
	}
	if err := p.SetCookies(cookies); err != nil {
		return tools.Result{}, fmt.Errorf("set cookies: %w", err)
	}
	return tools.Result{Output: map[string]any{"set": len(cookies)}}, nil
}

// ── 8. browser_eval (escape hatch — JS di halaman) ──────────────────────────
type browserEvalTool struct{}

func (browserEvalTool) Name() string       { return "browser_eval" }
func (browserEvalTool) Capability() string { return browserCap }
func (browserEvalTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Jalankan JS di halaman saat ini, balikin hasilnya (string). Escape hatch buat baca/aksi yg ga kecover tool lain.",
		Params:      []tools.Param{{Name: "js", Type: tools.ParamString, Description: "ekspresi JS, mis. '() => document.title'", Required: true}},
		Returns:     "{result}",
	}
}
func (browserEvalTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	js, _ := args["js"].(string)
	if strings.TrimSpace(js) == "" {
		return tools.Result{}, fmt.Errorf("js required")
	}
	p, err := getPage()
	if err != nil {
		return tools.Result{}, err
	}
	p = pageTimeout(p, 20*time.Second)
	res, err := p.Eval(js)
	if err != nil {
		return tools.Result{}, fmt.Errorf("eval: %w", err)
	}
	return tools.Result{Output: map[string]any{"result": res.Value.String()}}, nil
}

// ── lifecycle hygiene: close + idle-reaper (anti zombie chromium numpuk) ─────
// closeBrowser — tutup browser + free resource. Idempoten. Cuma nutup instance yg
// go-rod launch (BUKAN Chrome owner — beda proses/profil).
func closeBrowser() {
	brMu.Lock()
	defer brMu.Unlock()
	if brInst != nil {
		_ = brInst.Close()
		brInst, brPage = nil, nil
	}
}

// startBrowserReaper — jaring pengaman: browser nganggur >30 menit di-close otomatis.
// Skill WAJIB browser_close tiap tugas kelar; ini backstop kalau lupa. Dijalanin sekali
// (brReaper sync.Once) pas browser pertama dipakai.
func startBrowserReaper() {
	go func() {
		t := time.NewTicker(2 * time.Minute)
		defer t.Stop()
		for range t.C {
			brMu.Lock()
			idle := brInst != nil && time.Since(brLastUsed) > 30*time.Minute
			brMu.Unlock()
			if idle {
				closeBrowser()
			}
		}
	}()
}

// browser_close — tutup browser eksplisit. WAJIB dipanggil tiap tugas browser kelar.
type browserCloseTool struct{}

func (browserCloseTool) Name() string       { return "browser_close" }
func (browserCloseTool) Capability() string { return browserCap }
func (browserCloseTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Tutup browser + bebasin resource. WAJIB dipanggil tiap tugas browser kelar (anti zombie chromium numpuk). Browser auto-launch lagi pas tool browser dipakai berikutnya.",
		Params:      nil,
		Returns:     "{closed: true}",
	}
}
func (browserCloseTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	closeBrowser()
	return tools.Result{Output: map[string]any{"closed": true}}, nil
}
