package domain

import (
	"context"
	"errors"
	"pb_launcher/helpers/process"
	"pb_launcher/internal/launcher/domain/repositories"
	"sync"
)

type LauncherManager struct {
	sync.RWMutex
	repository repositories.ServiceRepository
	services   map[string]*process.Process
}

func NewLauncherManager(repository repositories.ServiceRepository) *LauncherManager {
	return &LauncherManager{
		repository: repository,
		services:   make(map[string]*process.Process),
	}
}

func (lm *LauncherManager) Run(_ context.Context) error {

	return nil
}

func (lm *LauncherManager) Stop() error {
	lm.Lock()
	defer lm.Unlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var combinedErr error

	collectError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		combinedErr = errors.Join(combinedErr, err)
	}

	for _, service := range lm.services {
		wg.Add(1)
		go func(s *process.Process) {
			defer wg.Done()
			if !s.IsRunning() {
				return
			}
			if err := s.Stop(); err != nil {
				collectError(err)
			}
		}(service)
	}

	wg.Wait()
	return combinedErr
}
