# Downtomate — Perencanaan Project Final

> Background daemon Go yang mengorganisir folder Downloads secara otomatis,
> seringan mungkin, tanpa ganggu aktivitas.

---

## Stack Final (dikunci)

| Komponen | Pilihan | Alasan |
|---|---|---|
| Bahasa | Go 1.22+ | Binary tunggal, tidak butuh runtime, goroutine ringan |
| File watcher | `fsnotify v1.10` | Event-driven, 0% CPU saat idle |
| AI runtime | Ollama + `keep_alive=0s` | Model unload otomatis setelah dipakai |
| AI model | `qwen2.5:0.5b` | 392MB disk, ~500MB RAM hanya saat aktif |
| Config | `config.json` | Sederhana, mudah diedit user |
| Log | File teks biasa | Tidak perlu library eksternal |

**Target resource:**
- Idle: ~15MB RAM, 0% CPU
- Proses Tier 1/2: ~20MB RAM, <1% CPU, selesai <50ms
- Proses Tier 3 (AI): spike ~500MB RAM selama 3–5 detik, lalu kembali 15MB

---

## Struktur Folder

```
downtomate/
├── main.go                  ← entry point, start semua komponen
├── go.mod
├── config.json              ← konfigurasi user
├── downtomate.log           ← log aktivitas (auto-dibuat)
│
├── config/
│   └── config.go            ← baca & validasi config.json
│
├── watcher/
│   └── watcher.go           ← fsnotify + debounce 1500ms
│
├── engine/
│   ├── engine.go            ← orkestrator: terima file, jalankan tier 1→2→3
│   ├── matcher.go           ← Tier 1 (ekstensi) + Tier 2 (keyword scoring)
│   └── ai.go                ← Tier 3: kirim ke Ollama, parse respons
│
├── mover/
│   └── mover.go             ← buat subfolder + pindahkan file (dengan dry-run)
│
└── logger/
    └── logger.go            ← append ke downtomate.log
```

---

## Skema `config.json` (final)

```json
{
  "watch_directory": "C:/Users/NamaUser/Downloads",
  "dry_run": false,
  "debounce_ms": 1500,
  "worker_count": 1,
  "ai": {
    "enabled": false,
    "ollama_url": "http://localhost:11434",
    "model": "qwen2.5:0.5b",
    "max_chars": 400,
    "keep_alive": "0s"
  },
  "ignore_extensions": [".tmp", ".part", ".crdownload", ".download"],
  "rules": [
    {
      "folder": "Software_dan_Installer",
      "extensions": [".exe", ".msi", ".dmg", ".pkg", ".deb", ".rpm", ".appimage"]
    },
    {
      "folder": "Video",
      "extensions": [".mp4", ".mkv", ".avi", ".mov", ".webm", ".flv"]
    },
    {
      "folder": "Audio",
      "extensions": [".mp3", ".flac", ".wav", ".aac", ".ogg", ".m4a"]
    },
    {
      "folder": "Gambar",
      "extensions": [".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".raw"]
    },
    {
      "folder": "Arsip",
      "extensions": [".zip", ".rar", ".7z", ".tar.gz", ".gz", ".bz2"]
    },
    {
      "folder": "Font",
      "extensions": [".ttf", ".otf", ".woff", ".woff2"]
    },
    {
      "folder": "Disk_Image",
      "extensions": [".iso", ".img"]
    },
    {
      "folder": "Torrent",
      "extensions": [".torrent"]
    },
    {
      "folder": "Dokumen_Kuliah",
      "extensions": [".pdf", ".docx", ".doc"],
      "keywords": [
        {"word": "tugas",   "weight": 5},
        {"word": "PBO",     "weight": 10},
        {"word": "UTS",     "weight": 8},
        {"word": "UAS",     "weight": 8},
        {"word": "modul",   "weight": 5},
        {"word": "praktikum","weight": 7}
      ],
      "keyword_threshold": 5,
      "ai_prompt": "Dokumen ini tentang topik apa? Jawab HANYA satu kata (contoh: Kuliah, Keuangan, Hukum, Resep, Teknik, Lainnya)."
    },
    {
      "folder": "Keuangan",
      "extensions": [".pdf", ".xlsx", ".csv"],
      "keywords": [
        {"word": "invoice",  "weight": 10},
        {"word": "struk",    "weight": 10},
        {"word": "receipt",  "weight": 10},
        {"word": "tagihan",  "weight": 8},
        {"word": "pajak",    "weight": 7},
        {"word": "laporan_keuangan", "weight": 9}
      ],
      "keyword_threshold": 7
    }
  ]
}
```

---

## Tanggung Jawab Tiap File

### `main.go`
- Load config
- Setup logger
- Jalankan watcher di goroutine
- Blokir dengan `select {}` (daemon tetap hidup)
- Handle `SIGINT`/`SIGTERM` untuk shutdown bersih

### `config/config.go`
```go
type Config struct {
    WatchDirectory   string   `json:"watch_directory"`
    DryRun           bool     `json:"dry_run"`
    DebounceMs       int      `json:"debounce_ms"`
    WorkerCount      int      `json:"worker_count"`
    AI               AIConfig `json:"ai"`
    IgnoreExtensions []string `json:"ignore_extensions"`
    Rules            []Rule   `json:"rules"`
}

type AIConfig struct {
    Enabled    bool   `json:"enabled"`
    OllamaURL  string `json:"ollama_url"`
    Model      string `json:"model"`
    MaxChars   int    `json:"max_chars"`
    KeepAlive  string `json:"keep_alive"`
}

type Rule struct {
    Folder            string    `json:"folder"`
    Extensions        []string  `json:"extensions"`
    Keywords          []Keyword `json:"keywords"`
    KeywordThreshold  int       `json:"keyword_threshold"`
    AIPrompt          string    `json:"ai_prompt"`
}

type Keyword struct {
    Word   string `json:"word"`
    Weight int    `json:"weight"`
}
```

### `watcher/watcher.go`
- Inisialisasi fsnotify watcher pada `watch_directory`
- Hanya proses event `CREATE` dan `RENAME` (abaikan WRITE, CHMOD)
- Debounce per file: reset timer 1500ms setiap event masuk untuk file yang sama
- Setelah debounce selesai, kirim path file ke channel engine
- Filter: skip hidden files (awalan `.`), skip ekstensi di `ignore_extensions`
- Skip subfolder yang dibuat oleh downtomate sendiri (cegah infinite loop)

```go
// Pseudocode debounce
timers := map[string]*time.Timer{}

onEvent(path):
    if timer exists: timer.Stop()
    timers[path] = time.AfterFunc(1500ms, func() {
        fileChan <- path
        delete(timers, path)
    })
```

### `engine/engine.go`
- Baca dari `fileChan` dengan worker pool (default: 1 worker)
- Untuk setiap file: jalankan `matcher.Classify(file, rules)`
- Jika ada match: panggil `mover.Move(src, destFolder)`
- Jika tidak ada match: log sebagai "tidak terklasifikasi"

### `engine/matcher.go`
```
Tier 1 — Extension match:
  - Ambil ekstensi file (lowercase)
  - Loop rules, cari rule yang ekstensinya cocok
  - Jika hanya 1 rule cocok → return rule itu (terminal)
  - Jika >1 rule cocok (gateway) → lanjut Tier 2

Tier 2 — Keyword scoring:
  - Ambil nama file (lowercase, tanpa ekstensi)
  - Hitung skor tiap rule yang lolos Tier 1
  - Rule dengan skor tertinggi >= keyword_threshold → return rule itu
  - Jika tidak ada yang mencapai threshold → lanjut Tier 3

Tier 3 — AI (hanya jika ai.enabled=true dan rule punya ai_prompt):
  - Ekstrak teks file (max 400 karakter)
  - Kirim ke Ollama
  - Parse respons → cocokkan ke nama folder
```

### `engine/ai.go`
- Fungsi `ExtractText(path) string`:
  - `.txt`, `.md` → baca langsung, ambil 400 char pertama
  - `.pdf` → pakai `ledongthuc/pdfcpu` atau panggil subprocess `pdftotext`
  - `.docx` → pakai `nguyenthenguyen/docconv`
  - File lain → return `""` (tidak diekstrak)
- Fungsi `Classify(text, prompt, cfg AIConfig) string`:
  - POST ke `ollama_url/api/generate`
  - Body: `{model, prompt+text, stream:false, keep_alive:"0s"}`
  - Return respons model (satu kata kategori)
  - Timeout: 30 detik
  - Jika error/timeout → return `""` (fallback ke Uncategorized)

### `mover/mover.go`
- `Move(src, watchDir, folderName, dryRun)`:
  - Buat path tujuan: `watchDir/folderName/namafile`
  - Jika `dryRun=true`: hanya log, tidak benar-benar pindah
  - Jika folder belum ada: `os.MkdirAll()`
  - Jika nama file sudah ada di tujuan: tambahkan suffix `_1`, `_2`, dst.
  - `os.Rename(src, dest)` — atomic di filesystem yang sama

### `logger/logger.go`
- Buka `downtomate.log` dengan mode append
- Format: `[2024-11-15 14:32:01] MOVED  tugas3_PBO.pdf → Dokumen_Kuliah/ (tier:2, score:15)`
- Format dry-run: `[2024-11-15 14:32:01] DRYRUN tugas3_PBO.pdf → Dokumen_Kuliah/`
- Format skip: `[2024-11-15 14:32:01] SKIP   .crdownload file diabaikan`

---

## Trik Optimasi Resource (Penting)

### 1. `keep_alive: "0s"` pada Ollama
Secara default Ollama menyimpan model di RAM selama 5 menit.
Dengan `keep_alive: "0s"`, model langsung di-unload setelah respons.
Hasilnya: RAM kembali ke ~15MB setelah AI selesai.

### 2. Worker count = 1
Hanya 1 file diproses sekaligus. Cegah lonjakan CPU saat banyak file masuk
sekaligus (misal: extract arsip besar).

### 3. Debounce 1500ms
Download manager menulis file secara bertahap. Tanpa debounce, fsnotify
bisa fire 50+ event untuk satu file. Debounce memastikan file diproses
sekali saja setelah download selesai.

### 4. Baca maksimal 400 karakter untuk AI
File PDF bisa ratusan halaman. AI hanya butuh paragraf pertama untuk
menentukan kategori. Ini potong waktu ekstraksi dan ukuran request.

### 5. Skip binary files di Tier 3
File `.jpg`, `.mp4`, `.exe`, dll tidak perlu dibaca isinya.
Matcher langsung return dari Tier 1 tanpa pernah menyentuh Tier 3.

### 6. Tidak ada polling
fsnotify murni event-driven dari OS. Tidak ada loop yang cek file
setiap N detik. CPU usage benar-benar 0% saat tidak ada file baru.

---

## Dependencies Go (`go.mod`)

```
module downtomate

go 1.22.0

require (
    github.com/fsnotify/fsnotify       v1.10.1   // file watching
    golang.org/x/sys                   v0.13.0   // dep fsnotify
    // Opsional, hanya jika AI diaktifkan:
    // github.com/ledongthuc/pdf       v0.0.0    // baca PDF
    // github.com/nguyenthenguyen/docconv v0.1.0 // baca DOCX
)
```

Untuk MVP (tanpa AI), hanya butuh `fsnotify` dan `golang.org/x/sys`.
Tambahkan PDF/DOCX reader nanti saat Tier 3 mau diaktifkan.

---

## Urutan Pengerjaan

```
Fase 1 — Inti (bisa dipakai)
  [1] config/config.go        ← struct + JSON parser
  [2] logger/logger.go        ← append log ke file
  [3] mover/mover.go          ← mkdir + rename + dry-run
  [4] engine/matcher.go       ← tier 1 + tier 2 (tanpa AI dulu)
  [5] engine/engine.go        ← worker pool + orchestrate
  [6] watcher/watcher.go      ← fsnotify + debounce
  [7] main.go                 ← rangkai semua

Fase 2 — Kenyamanan
  [8]  Dry-run mode           ← sudah ada di mover, expose di config
  [9]  Log rotation           ← batasi downtomate.log max 5MB
  [10] Reload config          ← watch config.json, reload tanpa restart

Fase 3 — AI (opsional, aktifkan sendiri)
  [11] engine/ai.go           ← ollama client + text extractor
  [12] Test dengan qwen2.5:0.5b
```

---

## Cara Install & Jalankan

```bash
# Install Ollama (hanya jika mau pakai AI)
# https://ollama.com/download

# Pull model terkecil
ollama pull qwen2.5:0.5b

# Build downtomate
cd downtomate
go build -o downtomate.exe .

# Edit config.json sesuai folder download kamu
# Jalankan
./downtomate.exe

# Jalankan dengan dry-run dulu (aman, tidak pindah file)
# Set "dry_run": true di config.json
```

### Jalankan sebagai startup (Windows)
Buat shortcut `downtomate.exe` di:
`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`

### Jalankan sebagai startup (Linux)
```ini
# ~/.config/systemd/user/downtomate.service
[Unit]
Description=Downtomate File Organizer

[Service]
ExecStart=/home/user/downtomate/downtomate
Restart=on-failure

[Install]
WantedBy=default.target
```
```bash
systemctl --user enable downtomate
systemctl --user start downtomate
```

---

## Contoh Output Log

```
[2024-11-15 09:01:02] START  Memantau: C:/Users/User/Downloads
[2024-11-15 09:14:33] MOVED  VLC_3.0.21_win64.exe → Software_dan_Installer/ (tier:1)
[2024-11-15 09:22:11] MOVED  tugas3_PBO_revisi.pdf → Dokumen_Kuliah/ (tier:2, score:15)
[2024-11-15 10:05:44] MOVED  lagu_favorit.mp3 → Audio/ (tier:1)
[2024-11-15 10:31:02] AI     laporan_akhir.pdf → Keuangan/ (tier:3, model:qwen2.5:0.5b)
[2024-11-15 11:00:00] SKIP   .crdownload diabaikan (ekstensi di ignore list)
[2024-11-15 11:00:45] NOMATCH random_file_tanpa_rule.xyz → _Uncategorized/
```
