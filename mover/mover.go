package mover

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"downtomate/logger"
)

func Move(src, watchDir, folderName string, dryRun bool) error {
	fileName := filepath.Base(src)
	targetDir := filepath.Join(watchDir, folderName)

	if dryRun {
		logger.Log("DRYRUN", fmt.Sprintf("%s -> %s/", fileName, folderName))
		return nil
	}

	// Create subfolder if it doesn't exist
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Handle duplicate filename
	dest := filepath.Join(targetDir, fileName)
	if _, err := os.Stat(dest); err == nil {
		dest = getUniquePath(targetDir, fileName)
	}

	// Atomic rename
	err := os.Rename(src, dest)
	if err != nil {
		// Fallback to Copy + Delete if Rename fails (e.g. cross-device)
		err = copyAndDelete(src, dest)
	}

	if err == nil {
		logger.Log("MOVED", fmt.Sprintf("%s -> %s/", fileName, folderName))
	}

	return err
}

func getUniquePath(dir, fileName string) string {
	ext := filepath.Ext(fileName)
	nameOnly := strings.TrimSuffix(fileName, ext)
	counter := 1
	for {
		newPath := filepath.Join(dir, fmt.Sprintf("%s_%d%s", nameOnly, counter, ext))
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

func copyAndDelete(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	source.Close()
	return os.Remove(src)
}
