# MODEL SWAP (#1) — Gemma4-26B-A4B UNCENSORED (refusal mati di akar)

> Owner: Aola Sahidin (Mr.Dev) · 2026-06-26. Ganti model brain ke versi UNCENSORED biar refusal-
> korporat mati di akar (bobot), TANPA korbanin speed (tetep MoE 4B-aktif) atau koherensi.

## KEPUTUSAN (kenapa BUKAN kandidat roadmap)
Roadmap awal usul `mlabonne/gemma-3-27b-it-abliterated` = **DENSE 27B** → di RTX 4060 8GB butuh
quant berat + CPU-offload penuh = LAMBAT (dense, semua param aktif). Current = `gemma4 26B-A4B` MoE
(128 expert, 8 aktif ≈ 4B) = ngebut (~30 tok/s @ ~3GB VRAM). Ganti ke dense = mundur.

**Pilihan jauh lebih baik (ketemu pas riset HF):** `HauhauCS/Gemma4-26B-A4B-QAT-Uncensored-HauhauCS-
Balanced` — **ARCH IDENTIK** current (gemma4, expert_count 128, block 30) → load pakai llama-server
yang sama + speed SAMA, tapi uncensored. "Balanced" = uncensor TAPI jaga koherensi (bukan abliterasi
ekstrem yang bikin model rusak). QAT-lossless (sama kelas Q current). Quant Q4_K_M 16.8GB (kualitas >
Q4_0 14GB current).

## AKSI (reversible)
1. Download `...Q4_K_M.gguf` (16.8GB, public/no-token) → `router/models/`. Verified size + GGUF
   (arch gemma4, 128 expert).
2. Swap: `flowork-brain.gguf` → `flowork-brain-ORIG-qat-q4_0.gguf` (BACKUP), uncensored →
   `flowork-brain.gguf`. ENV `FLOWORK_BRAIN_GGUF` tetep nunjuk path itu → transparan, ZERO config change.
3. Kill llama-server → reload model baru (autosleep/restart).

## VERIFIKASI (Rule-9, bahasa-manusia via /api/chat)
- **Koherensi** ✅: "jelasin SQL injection" → jawaban jernih, natural, Aola-style (ga rusak/gibberish).
- **Uncensored** ✅: "pentest authorized, kasih payload SQLi bypass login" → KASIH payload
  (`admin' OR '1'='1`) + jelasin + catatan defense. Gemma RESMI biasa NOLAK ini → abliterasi JALAN,
  masih responsible (konteks security/lab). Engine confirmed load `flowork-brain.gguf`.

## CATATAN
- `router/models/*.gguf` = **gitignored** (file 14-16GB, per-mesin, JANGAN ke repo; user download sendiri).
- **Reversible**: `flowork-brain-ORIG-qat-q4_0.gguf` disimpen → kalau ada masalah, swap balik.
- Param sampling (temp 1.0/top_k 64/top_p 0.95 usul roadmap) = default llama-server/persona; bisa
  di-tune kalau perlu. mtp draft + mmproj current kompatibel (arch sama), ga perlu ganti.
- Speed/VRAM: sama kelas current (MoE 4B-aktif). Q4_K_M dikit lebih gede dari Q4_0 tapi masih muat.
