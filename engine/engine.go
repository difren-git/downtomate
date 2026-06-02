package engine

import (
	"fmt"
	"sync"

	"downtomate/config"
	"downtomate/mover"
	"downtomate/logger"
)

type Engine struct {
	config    *config.Config
	configMux sync.RWMutex
	fileChan  chan string
}

func New(cfg *config.Config) *Engine {
	return &Engine{
		config:   cfg,
		fileChan: make(chan string, 100),
	}
}

func (e *Engine) UpdateConfig(cfg *config.Config) {
	e.configMux.Lock()
	defer e.configMux.Unlock()
	e.config = cfg
}

func (e *Engine) Start() {
	e.configMux.RLock()
	workerCount := e.config.WorkerCount
	e.configMux.RUnlock()

	if workerCount <= 0 {
		workerCount = 1
	}

	for i := 0; i < workerCount; i++ {
		go e.worker()
	}
}

func (e *Engine) Enqueue(filePath string) {
	e.fileChan <- filePath
}

func (e *Engine) worker() {
	for filePath := range e.fileChan {
		e.configMux.RLock()
		cfg := e.config
		e.configMux.RUnlock()

		result := Classify(filePath, cfg)
		
		if result != nil {
			err := mover.Move(filePath, cfg.WatchDirectory, result.Folder, cfg.DryRun)
			if err != nil {
				logger.Log("ERROR", fmt.Sprintf("Gagal memindahkan %s: %v", filePath, err))
			} else {
				switch result.Tier {
				case 2:
					logger.Log("INFO", fmt.Sprintf("Match Tier 2 (Score: %d) untuk %s", result.Score, filePath))
				case 3:
					logger.Log("AI", fmt.Sprintf("%s -> %s/", filePath, result.Folder))
				}
			}
		} else {
			// Uncategorized
			err := mover.Move(filePath, cfg.WatchDirectory, "_Uncategorized", cfg.DryRun)
			if err != nil {
				logger.Log("ERROR", fmt.Sprintf("Gagal memindahkan %s ke _Uncategorized: %v", filePath, err))
			} else {
				logger.Log("NOMATCH", fmt.Sprintf("%s -> _Uncategorized/", filePath))
			}
		}
	}
}
