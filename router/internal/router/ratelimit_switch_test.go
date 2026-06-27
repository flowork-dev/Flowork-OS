package router

import "testing"

// switch FLOWORK_RL_MAX_RETRY: env → nilai, default 6, clamp [0,20].
func TestMaxRateLimitRetries_Switch(t *testing.T) {
	cases := []struct {
		env  string
		want int
	}{
		{"", 6},      // unset → default (perilaku lama)
		{"0", 0},     // 0 = lompat fallback INSTAN (no retry)
		{"2", 2},     // fast fallback
		{"20", 20},   // batas atas
		{"21", 6},    // di luar range → default
		{"-1", 6},    // negatif → default
		{"abc", 6},   // bukan angka → default
	}
	for _, c := range cases {
		t.Setenv("FLOWORK_RL_MAX_RETRY", c.env)
		if got := maxRateLimitRetries(); got != c.want {
			t.Errorf("env=%q → got %d, want %d", c.env, got, c.want)
		}
	}
}
