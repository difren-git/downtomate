package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var (
	logFile    *os.File
	logPath    string
	logMux     sync.Mutex
	maxLogSize int64 = 5 * 1024 * 1024 // 5MB
)

func Init(path string) error {
	logPath = path
	var err error
	logFile, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return nil
}

func Log(action, message string) {
	logMux.Lock()
	defer logMux.Unlock()

	rotateIfNeeded()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %-6s %s\n", timestamp, action, message)
	
	// Print to console
	fmt.Print(logLine)
	
	// Write to file
	if logFile != nil {
		logFile.WriteString(logLine)
	}
}

func rotateIfNeeded() {
	if logFile == nil {
		return
	}

	info, err := logFile.Stat()
	if err != nil {
		return
	}

	if info.Size() > maxLogSize {
		logFile.Close()
		
		oldPath := logPath + ".old"
		os.Remove(oldPath) // Remove existing .old if any
		os.Rename(logPath, oldPath)

		var err error
		logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Gagal rotate log: %v\n", err)
		}
	}
}

func Close() {
	logMux.Lock()
	defer logMux.Unlock()
	if logFile != nil {
		logFile.Close()
	}
}

func Fatal(err error) {
	Log("FATAL", err.Error())
	log.Fatal(err)
}
