package mover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMove(t *testing.T) {
	// Setup Lingkungan Pura-pura (Virtual Environment)
	tmpDir := t.TempDir()
	watchDir := filepath.Join(tmpDir, "Downloads")
	os.Mkdir(watchDir, 0755)

	// Buat file sumber bohongan
	srcFile := filepath.Join(tmpDir, "file_tes.txt")
	os.WriteFile(srcFile, []byte("Halo ini file tes"), 0644)

	// Uji 1: Pemindahan normal (sukses membuat folder baru)
	err := Move(srcFile, watchDir, "Dokumen", false)
	if err != nil {
		t.Fatalf("Move gagal: %v", err)
	}

	expectedDest1 := filepath.Join(watchDir, "Dokumen", "file_tes.txt")
	if _, err := os.Stat(expectedDest1); os.IsNotExist(err) {
		t.Errorf("File gagal dipindahkan ke %s", expectedDest1)
	}

	// Uji 2: Penanganan Nama Sama (Duplicate Handling)
	// Kita buat file baru di tempat asal dengan nama yang sama persis
	os.WriteFile(srcFile, []byte("Ini file tes kedua dengan nama sama"), 0644)
	
	err = Move(srcFile, watchDir, "Dokumen", false)
	if err != nil {
		t.Fatalf("Move duplicate gagal: %v", err)
	}

	// Seharusnya file yang baru ini diubah namanya jadi _1
	expectedDest2 := filepath.Join(watchDir, "Dokumen", "file_tes_1.txt")
	if _, err := os.Stat(expectedDest2); os.IsNotExist(err) {
		t.Errorf("Penanganan duplikat gagal, file _1 tidak ditemukan di %s", expectedDest2)
	}
}
