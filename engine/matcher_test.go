package engine

import (
	"testing"
	"downtomate/config"
)

func TestClassify(t *testing.T) {
	// Siapkan konfigurasi pura-pura untuk tes
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Folder: "Gambar_App",
				Mode: "Ekstensi",
				Extensions: []string{".jpg", ".png"},
			},
			{
				Folder: "Tugas_Sekolah",
				Mode: "Keyword",
				Extensions: []string{".pdf", ".docx"},
				Keywords: []config.Keyword{
					{Word: "tugas", Weight: 5},
				},
				KeywordThreshold: 5,
			},
		},
	}

	tests := []struct {
		name         string
		fileName     string
		expected     string
		expectedTier int
	}{
		{
			name:         "Mode Ekstensi Benar",
			fileName:     "C:/Downloads/foto_liburan.jpg",
			expected:     "Gambar_App",
			expectedTier: 1,
		},
		{
			name:         "Mode Keyword Cocok",
			fileName:     "C:/Downloads/tugas_matematika.pdf",
			expected:     "Tugas_Sekolah",
			expectedTier: 2,
		},
		{
			name:         "Ekstensi Cocok, Keyword Tidak Ada (Harus Gagal)",
			fileName:     "C:/Downloads/buku_bacaan.pdf", // Tidak ada kata "tugas"
			expected:     "", // Artinya tidak cocok dan akan masuk Uncategorized
			expectedTier: 0,
		},
		{
			name:         "Ekstensi Salah (Harus Gagal)",
			fileName:     "C:/Downloads/lagu.mp3",
			expected:     "",
			expectedTier: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Classify(tt.fileName, cfg)
			
			if tt.expected == "" {
				if result != nil {
					t.Errorf("Diharapkan gagal (nil), tapi malah cocok dengan folder %s", result.Folder)
				}
			} else {
				if result == nil {
					t.Errorf("Diharapkan cocok dengan folder %s, tapi gagal (nil)", tt.expected)
				} else if result.Folder != tt.expected || result.Tier != tt.expectedTier {
					t.Errorf("Diharapkan folder %s (Tier %d), tapi dapat folder %s (Tier %d)", 
						tt.expected, tt.expectedTier, result.Folder, result.Tier)
				}
			}
		})
	}
}
