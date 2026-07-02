// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// vindex_ready_ext.go — sibling NON-frozen (Rule 7): expose kesiapan index vektor
// semantic ke luar package, buat seam enrichment-selektif di router. Ga nyentuh
// frozen semantic.go / semantic_threshold_ext.go. Dihapus = seam fallback perilaku
// lama (fail-open), inti tetep jalan.

package brain

// VectorIndexReady — true kalau index vektor semantic udah kebangun & siap dipakai.
// Dipakai seam enrichment selektif: index belum siap → caller fallback ke retrieve
// lama (SemanticRetrieve) biar perilaku ga berubah di mesin tanpa index.
func VectorIndexReady() bool { return loadVIndex() != nil }
