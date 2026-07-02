// allow_seam.go — papan colokan (Pola A) allowlist endpoint loopback tanpa-sesi.
// 📄 Dok: FLowork_os/lock/approval-gate.md (bagian seam auth)
//
// AKAR (2026-07-02): allowlist isPublicPath = switch hardcoded di handlers.go
// (FROZEN) → tiap endpoint loopback baru maksa buka file beku (arsitektur cacat,
// Rule #7). Papan ini nutup itu: endpoint baru dicolok dari file sibling _ext.go
// (deletable) via RegisterLoopbackPublic.
//
// INVARIAN KEAMANAN (dipaksa TERPUSAT di sini, ext ga bisa bypass):
//   1. Cuma request LOOPBACK (isLocalRequest — TCP peer, ga bisa di-spoof).
//   2. Cross-site browser tetep DITOLAK (isCrossSiteBrowser — anti drive-by).
//   3. Method wajib cocok daftar yang didaftarin.
// Jadi ext yang salah/typo PALING BURUK cuma ngebuka endpoint ke owner-local —
// ga pernah ke remote. Semua ext dihapus → papan kosong = perilaku lama persis.
package floworkauth

import (
	"net/http"
	"strings"
	"sync"
)

type loopbackAllowEntry struct {
	path    string          // exact match (case-sensitive, kayak switch aslinya)
	methods map[string]bool // kosong = semua method
}

var (
	loopbackAllowMu   sync.RWMutex
	loopbackAllowList []loopbackAllowEntry
)

// RegisterLoopbackPublic — daftarin path yang boleh diakses TANPA sesi, KHUSUS
// loopback (curl/agent lokal). methods kosong = semua method.
func RegisterLoopbackPublic(path string, methods ...string) {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return
	}
	m := map[string]bool{}
	for _, v := range methods {
		if v = strings.ToUpper(strings.TrimSpace(v)); v != "" {
			m[v] = true
		}
	}
	loopbackAllowMu.Lock()
	loopbackAllowList = append(loopbackAllowList, loopbackAllowEntry{path: path, methods: m})
	loopbackAllowMu.Unlock()
}

// loopbackAllowExt — dipanggil isPublicPath (handlers.go, frozen) sebagai
// fallback terakhir. Default (papan kosong) = false → perilaku lama utuh.
func loopbackAllowExt(path string, r *http.Request) bool {
	loopbackAllowMu.RLock()
	defer loopbackAllowMu.RUnlock()
	for _, e := range loopbackAllowList {
		if e.path != path {
			continue
		}
		if len(e.methods) > 0 && !e.methods[r.Method] {
			continue
		}
		// Invarian terpusat — ga peduli ext-nya nulis apa.
		if isCrossSiteBrowser(r) {
			return false
		}
		return isLocalRequest(r)
	}
	return false
}
