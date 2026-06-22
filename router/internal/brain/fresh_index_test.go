package brain

import "testing"

// mergeFresh HARUS additif & 0-regresi: fresh kosong → hasil PERSIS main (urutan kejaga).
func TestMergeFresh_EmptyFreshIsNoop(t *testing.T) {
	main := []Snippet{
		{DrawerID: "a", Score: 0.9},
		{DrawerID: "b", Score: 0.5},
	}
	got := mergeFresh(main, nil, 6)
	if len(got) != 2 || got[0].DrawerID != "a" || got[1].DrawerID != "b" {
		t.Fatalf("fresh kosong harus no-op (identik main), dapat %+v", got)
	}
}

func TestMergeFresh_DedupAndSort(t *testing.T) {
	main := []Snippet{
		{DrawerID: "a", Score: 0.9},
		{DrawerID: "b", Score: 0.4},
	}
	fresh := []Snippet{
		{DrawerID: "c", Score: 0.7}, // baru, mestinya nyelip antara a & b
		{DrawerID: "a", Score: 0.6}, // duplikat id → JANGAN dobel (versi main menang)
	}
	got := mergeFresh(main, fresh, 6)
	if len(got) != 3 {
		t.Fatalf("harus 3 unik (a,b,c), dapat %d: %+v", len(got), got)
	}
	want := []string{"a", "c", "b"} // urut skor desc: 0.9, 0.7, 0.4
	for i, w := range want {
		if got[i].DrawerID != w {
			t.Fatalf("urutan salah di %d: dapat %s want %s (%+v)", i, got[i].DrawerID, w, got)
		}
	}
	// dedup: 'a' pakai skor main (0.9), bukan fresh (0.6).
	if got[0].Score != 0.9 {
		t.Fatalf("dedup harus pertahankan versi main (0.9), dapat %.2f", got[0].Score)
	}
}

func TestMergeFresh_CapLimit(t *testing.T) {
	main := []Snippet{{DrawerID: "a", Score: 0.9}, {DrawerID: "b", Score: 0.8}}
	fresh := []Snippet{{DrawerID: "c", Score: 0.7}}
	if got := mergeFresh(main, fresh, 2); len(got) != 2 {
		t.Fatalf("cap limit 2 gagal: %d", len(got))
	}
}

func TestFreshWhereIn(t *testing.T) {
	ph, args := freshWhereIn()
	if ph == "" || len(args) != len(freshMemTypes) {
		t.Fatalf("freshWhereIn salah: ph=%q args=%v", ph, args)
	}
}
