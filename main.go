package main

import (
	"encoding/json"
	"fmt"
	"os"

	"downtomate/config"
	"downtomate/engine"
	"downtomate/logger"
	"downtomate/ui"
	"downtomate/watcher"
)

func main() {
	configPath := "config.json"
	logPath := "downtomate.log"

	logger.Init(logPath)
	defer logger.Close()

	showNow := false
	if !configFileExists(configPath) {
		createDefaultConfig(configPath)
		showNow = true // Tampilkan GUI saat baru pertama kali dibuat
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Fatal(fmt.Errorf("gagal memuat config: %w", err))
	}

	// 1. Inisialisasi Engine & Watcher
	eng := engine.New(cfg)
	eng.Start()

	w := watcher.New(cfg, func(path string) {
		eng.Enqueue(path)
	})

	if err := w.Start(); err != nil {
		logger.Fatal(fmt.Errorf("gagal memulai watcher: %w", err))
	}
	logger.Log("START", fmt.Sprintf("Memantau: %s", cfg.WatchDirectory))

	// 2. Jalankan GUI dan serahkan kontrol event loop ke UI
	// GUI ini akan mem-blok kode sampai pengguna menekan "Keluar Sepenuhnya" di Tray.
	if err := ui.RunGUI(configPath, cfg, eng, w, showNow); err != nil {
		logger.Fatal(fmt.Errorf("gagal memuat GUI: %w", err))
	}

	// Cleanup
	w.Stop()
	logger.Log("STOP", "Aplikasi dihentikan dari tray.")
}

func configFileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func saveConfig(path string, c *config.Config) {
	data, _ := json.MarshalIndent(c, "", "  ")
	os.WriteFile(path, data, 0644)
}

func createDefaultConfig(path string) {
	defaultCfg := config.Config{
		WatchDirectory: "C:/",
		DryRun:         false,
		DebounceMs:     1500,
		WorkerCount:    1,
		AI: config.AIConfig{
			Enabled:      false,
			GeminiAPIKey: "",
			MaxChars:     2000,
		},
		IgnoreExtensions: []string{".tmp", ".part", ".crdownload", ".download"},
		Rules: []config.Rule{
			{Folder: "Images", Extensions: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}},
			{Folder: "Documents", Extensions: []string{".pdf", ".docx", ".txt"}},
			{Folder: "Installers", Extensions: []string{".exe", ".msi"}},
			{Folder: "Source_Code", Extensions: []string{".c", ".cpp", ".go", ".py", ".js"}, AIPrompt: "Apakah file ini berisi kode sumber pemrograman? Jawab 'Source_Code' jika iya."},
		},
	}
	saveConfig(path, &defaultCfg)
}
