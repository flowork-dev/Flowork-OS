package floworkauth

import (
	"net/http/httptest"
	"testing"
)

// Seam allowlist WAJIB maksa invarian terpusat: loopback-only, method cocok,
// cross-site browser ditolak — walau ext-nya didaftarin sembarangan.
func TestLoopbackAllowExt_Invariants(t *testing.T) {
	RegisterLoopbackPublic("/api/tes-seam", "GET")

	req := func(method, remote string, hdr map[string]string) bool {
		r := httptest.NewRequest(method, "/api/tes-seam", nil)
		r.RemoteAddr = remote
		for k, v := range hdr {
			r.Header.Set(k, v)
		}
		return loopbackAllowExt("/api/tes-seam", r)
	}

	if !req("GET", "127.0.0.1:5555", nil) {
		t.Error("GET loopback harus LOLOS")
	}
	if req("POST", "127.0.0.1:5555", nil) {
		t.Error("method di luar daftar harus DITOLAK")
	}
	if req("GET", "192.168.1.7:5555", nil) {
		t.Error("non-loopback harus DITOLAK (invarian terpusat)")
	}
	if req("GET", "127.0.0.1:5555", map[string]string{"Sec-Fetch-Site": "cross-site"}) {
		t.Error("cross-site browser harus DITOLAK (anti drive-by)")
	}
	if loopbackAllowExt("/api/ga-terdaftar", httptest.NewRequest("GET", "/api/ga-terdaftar", nil)) {
		t.Error("path yang ga didaftarin harus DITOLAK (papan kosong = perilaku lama)")
	}

	// Registrasi ngaco → diabaikan diam-diam (ext error ga ngerusak).
	RegisterLoopbackPublic("")
	RegisterLoopbackPublic("tanpa-slash")
}
