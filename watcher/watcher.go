package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"downtomate/config"
	"downtomate/logger"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	config    *config.Config
	configMux sync.RWMutex
	onFile    func(string)
	timers    map[string]*time.Timer
	timersMux sync.Mutex
	fswatcher *fsnotify.Watcher
	stopChan  chan struct{}
	
	isPaused  bool
	pauseMux  sync.RWMutex
}

func New(cfg *config.Config, onFile func(string)) *Watcher {
	return &Watcher{
		config:   cfg,
		onFile:   onFile,
		timers:   make(map[string]*time.Timer),
		stopChan: make(chan struct{}),
	}
}

func (w *Watcher) SetPaused(paused bool) {
	w.pauseMux.Lock()
	w.isPaused = paused
	w.pauseMux.Unlock()
}

func (w *Watcher) UpdateConfig(cfg *config.Config) {
	w.configMux.Lock()
	oldDir := w.config.WatchDirectory
	w.config = cfg
	newDir := cfg.WatchDirectory
	w.configMux.Unlock()

	if oldDir != newDir && w.fswatcher != nil {
		oldAbs, _ := filepath.Abs(oldDir)
		newAbs, _ := filepath.Abs(newDir)
		
		w.fswatcher.Remove(oldAbs)
		
		if _, err := os.Stat(newAbs); os.IsNotExist(err) {
			os.MkdirAll(newAbs, 0755)
		}
		w.fswatcher.Add(newAbs)
		logger.Log("CONFIG", "Watch directory diperbarui: "+newAbs)
	}
}

func (w *Watcher) Start() error {
	var err error
	w.fswatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	
	w.stopChan = make(chan struct{})

	go func() {
		for {
			select {
			case event, ok := <-w.fswatcher.Events:
				if !ok {
					return
				}
				
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Rename == fsnotify.Rename {
					w.handleEvent(event.Name)
				}
			case err, ok := <-w.fswatcher.Errors:
				if !ok {
					return
				}
				logger.Log("ERROR", err.Error())
			case <-w.stopChan:
				return
			}
		}
	}()

	w.configMux.RLock()
	dir := w.config.WatchDirectory
	w.configMux.RUnlock()

	absPath, _ := filepath.Abs(dir)
	err = w.fswatcher.Add(absPath)
	if err != nil {
		return err
	}

	return nil
}

func (w *Watcher) Stop() {
	if w.fswatcher != nil {
		close(w.stopChan)
		w.fswatcher.Close()
		w.fswatcher = nil
	}
}

func (w *Watcher) handleEvent(path string) {
	w.pauseMux.RLock()
	paused := w.isPaused
	w.pauseMux.RUnlock()
	
	if paused {
		return
	}

	// Skip directories
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return
	}

	fileName := filepath.Base(path)
	
	// Skip hidden files
	if strings.HasPrefix(fileName, ".") {
		return
	}

	w.configMux.RLock()
	cfg := w.config
	w.configMux.RUnlock()

	// Skip ignored extensions
	ext := strings.ToLower(filepath.Ext(fileName))
	for _, ie := range cfg.IgnoreExtensions {
		if strings.ToLower(ie) == ext {
			return
		}
	}

	// Debounce
	w.timersMux.Lock()
	if t, ok := w.timers[path]; ok {
		t.Stop()
	}
	
	w.timers[path] = time.AfterFunc(time.Duration(cfg.DebounceMs)*time.Millisecond, func() {
		w.timersMux.Lock()
		delete(w.timers, path)
		w.timersMux.Unlock()
		
		w.onFile(path)
	})
	w.timersMux.Unlock()
}
