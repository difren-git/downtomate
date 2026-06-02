package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Setup file JSON bohongan untuk test
	content := []byte(`{
		"watch_directory": "C:/Test",
		"rules": [
			{"folder": "Gambar", "extensions": [".jpg"]}
		]
	}`)
	tmpFile := filepath.Join(t.TempDir(), "test_config.json")
	os.WriteFile(tmpFile, content, 0644)

	// Uji jalankan fungsi LoadConfig
	cfg, err := LoadConfig(tmpFile)

	if err != nil {
		t.Fatalf("Gagal memuat config: %v", err)
	}

	if cfg.WatchDirectory != "C:/Test" {
		t.Errorf("WatchDirectory salah, dapat: %s", cfg.WatchDirectory)
	}

	if len(cfg.Rules) != 1 {
		t.Fatalf("Jumlah rules salah, harusnya 1, dapat: %d", len(cfg.Rules))
	}

	// Cek apakah nilai default otomatis terisi (jika tidak ada di JSON)
	if cfg.DebounceMs != 1500 {
		t.Errorf("Default DebounceMs gagal diterapkan, dapat: %d", cfg.DebounceMs)
	}
	if cfg.WorkerCount != 1 {
		t.Errorf("Default WorkerCount gagal diterapkan, dapat: %d", cfg.WorkerCount)
	}
}
