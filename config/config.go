package config

import (
	"encoding/json"
	"os"

	"github.com/joho/godotenv"
)

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
	Enabled      bool   `json:"enabled"`
	GeminiAPIKey string `json:"gemini_api_key"`
	MaxChars     int    `json:"max_chars"`
}

type Rule struct {
	Folder           string    `json:"folder"`
	Mode             string    `json:"mode"` // "Ekstensi", "Keyword", "AI"
	Extensions       []string  `json:"extensions"`
	Keywords         []Keyword `json:"keywords"`
	KeywordThreshold int       `json:"keyword_threshold"`
	AIPrompt         string    `json:"ai_prompt"`
}

type Keyword struct {
	Word   string `json:"word"`
	Weight int    `json:"weight"`
}

func LoadConfig(path string) (*Config, error) {
	// Coba load .env (abaikan error jika file tidak ada)
	godotenv.Load()

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	// Override Gemini API Key dari .env jika ada
	envAPIKey := os.Getenv("GEMINI_API_KEY")
	if envAPIKey != "" {
		cfg.AI.GeminiAPIKey = envAPIKey
	}

	// Default values
	if cfg.DebounceMs == 0 {
		cfg.DebounceMs = 1500
	}
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = 1
	}

	return &cfg, nil
}
