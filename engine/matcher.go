package engine

import (
	"path/filepath"
	"strings"

	"downtomate/config"
)

type MatchResult struct {
	Folder string
	Tier   int
	Score  int
}

func Classify(filePath string, cfg *config.Config) *MatchResult {
	rules := cfg.Rules
	fileName := filepath.Base(filePath)
	fileNameLower := strings.ToLower(fileName)
	ext := strings.ToLower(filepath.Ext(fileName))

	// Evaluasi aturan berurutan (prioritas atas ke bawah)
	for _, rule := range rules {
		// 1. Cek Ekstensi (Semua mode butuh filter ekstensi, kecuali jika kosong = semua ekstensi)
		extMatch := false
		if len(rule.Extensions) == 0 {
			extMatch = true // Jika kosong, berlaku untuk semua file
		} else {
			for _, e := range rule.Extensions {
				if strings.ToLower(e) == ext {
					extMatch = true
					break
				}
			}
		}

		if !extMatch {
			continue // Lanjut ke aturan berikutnya jika ekstensi tidak cocok
		}

		// 2. Evaluasi berdasarkan Mode yang dipilih
		mode := rule.Mode
		if mode == "" {
			mode = "Ekstensi" // Backward compatibility
		}

		if mode == "Ekstensi" {
			// Mode 1: Jika ekstensinya cocok (yang mana sudah dicek di atas), langsung pindahkan
			return &MatchResult{Folder: rule.Folder, Tier: 1}
		}

		if mode == "Keyword" {
			// Mode 2: Cari keyword pada nama file
			nameOnly := strings.TrimSuffix(fileNameLower, ext)
			score := 0
			for _, kw := range rule.Keywords {
				if strings.Contains(nameOnly, strings.ToLower(kw.Word)) {
					score += kw.Weight
				}
			}
			if score >= rule.KeywordThreshold && score > 0 {
				return &MatchResult{Folder: rule.Folder, Tier: 2, Score: score}
			}
		}

		if mode == "AI" && cfg.AI.Enabled {
			// Mode 3: Pakai AI untuk baca isi file
			if rule.AIPrompt != "" {
				text := ExtractText(filePath, cfg.AI.MaxChars)
				if text != "" {
					aiResponse := ClassifyWithAI(text, rule.AIPrompt, cfg.AI)
					if strings.Contains(strings.ToLower(aiResponse), strings.ToLower(rule.Folder)) {
						return &MatchResult{Folder: rule.Folder, Tier: 3}
					}
				}
			}
		}
	}

	return nil
}
