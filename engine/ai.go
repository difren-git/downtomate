package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"downtomate/config"
)

func ExtractText(path string, maxChars int) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	// For now, only support plain text and markdown
	if ext == ".txt" || ext == ".md" || ext == ".c" || ext == ".go" || ext == ".py" || ext == ".json" {
		f, err := os.Open(path)
		if err != nil {
			return ""
		}
		defer f.Close()

		buf := make([]byte, maxChars)
		n, _ := f.Read(buf)
		return string(buf[:n])
	}

	// TODO: Support PDF and DOCX in future phases
	return ""
}

func ClassifyWithAI(text, prompt string, cfg config.AIConfig) string {
	if cfg.GeminiAPIKey == "" {
		return ""
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", cfg.GeminiAPIKey)
	
	systemPrompt := fmt.Sprintf("Anda adalah asisten pengklasifikasi file. Analisis konten berikut dan jawab HANYA dengan satu kata yang merupakan nama folder yang tepat (misal: 'Kuliah_PBO', 'Keuangan', dll). Jika tidak yakin, jawab 'Uncategorized'.\n\nKriteria user: %s\n\nKonten:\n%s", prompt, text)
	
	requestBody, _ := json.Marshal(map[string]interface{}{
		"contents": []interface{}{
			map[string]interface{}{
				"parts": []interface{}{
					map[string]interface{}{
						"text": systemPrompt,
					},
				},
			},
		},
	})

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Parsing respons Gemini
	if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if content, ok := candidates[0].(map[string]interface{})["content"].(map[string]interface{}); ok {
			if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
				if text, ok := parts[0].(map[string]interface{})["text"].(string); ok {
					return strings.TrimSpace(text)
				}
			}
		}
	}

	return ""
}

