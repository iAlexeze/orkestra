package manager

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
)

const eventHandler = "event handler"

type Manager struct {
	components []domain.Komponent
	postStart  []postStart
	timeout    time.Duration
	done       chan struct{}
}

type postStart struct {
	hook func(context.Context)
	comp domain.Komponent
}

func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		timeout: timeout,
		done:    make(chan struct{}),
	}
}

func (m *Manager) Start(ctx context.Context) error {
	mCtx, mCancel := context.WithCancel(ctx)
	defer mCancel()

	fmt.Println("===============================")
	fmt.Println("STARTING MANAGER COMPONENTS...")
	for _, comp := range m.components {
		name := comp.Name()

		logger.Info().Msgf("[%s] starting...", name)
		if err := comp.Start(mCtx); err != nil {
			logger.Error().Err(err).Msgf("failed to start: %s", name)
			return err
		}
		utils.Sleep(1)
		logger.Info().Msgf("%s status: %v", name, utils.StatusOnline)
	}

	// Run post-start hooks (leader election goes here)
	logger.Info().Msg("Running post-start hooks...")
	for _, p := range m.postStart {
		go p.hook(mCtx)
	}

	logger.Info().Msg("✅ All services started successfully")

	// Display started components
	fmt.Println("===============================")
	fmt.Println("STARTED COMPONENTS:")
	n := 1
	for _, comp := range m.components {
		fmt.Printf("%d. %s\n", n, comp.Name())
		n++
	}

	for _, p := range m.postStart {
		fmt.Printf("%d. %s\n", n, p.comp.Name())
		n++
	}
	fmt.Println("===============================")

	m.gracefulShutdown(mCtx, mCancel)
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) {}

func (m *Manager) gracefulShutdown(ctx context.Context, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info().Msgf("recieved shutdown signal: %v", sig)
		cancel()

		// shutdown components
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), m.timeout)
		defer shutdownCancel()

		for _, comp := range utils.Reversed(m.components) {
			name := comp.Name()
			logger.Info().Msgf("shutting down: %s...", name)
			if comp != nil {
				// Special handling for event recorder - shut it down LAST
				if name == eventHandler {
					continue // Skip event handler for now
				}
				comp.Shutdown(shutdownCtx)
			}
			utils.Sleep(1)
			logger.Info().Msgf("%s status: %v", name, utils.StatusOffline)
		}

		ev := m.GetComponent(eventHandler)
		if ev != nil {
			ev.Shutdown(shutdownCtx)
		}

		logger.Info().Msg("✅ All services shut down gracefully")

		// Notify Wait() to terminate
		close(m.done)

	case <-ctx.Done():
		return
	}
}

// Register all components
func (m *Manager) Register(c []domain.Komponent) {
	fmt.Println("==================================")
	fmt.Println("REGISTERING MANAGER COMPONENTS...")
	for _, comp := range c {
		m.components = append(m.components, comp)
		logger.Info().Msgf("[%s] registered", comp.Name())
	}
	logger.Info().Msg("✅ All services registered successfully")

	// Display registered components
	fmt.Println("==================================")
	fmt.Println("REGISTERED COMPONENTS:")
	n := 1
	for _, comp := range m.components {
		fmt.Printf("%d. %s\n", n, comp.Name())
		n++
	}
}

// GetComponent returns a component if present
func (m *Manager) GetComponent(name string) domain.Komponent {
	for _, comp := range m.components {
		if comp.Name() == name {
			return comp
		}
	}
	return nil
}

// AddPostStartHook: for services that need to start after manager has started
func (m *Manager) AddPostStartHook(comp domain.Komponent, hook func(context.Context)) {
	m.postStart = append(m.postStart, postStart{
		hook: hook,
		comp: comp,
	})
}

// Listening to done channel
func (m *Manager) Wait() {
	<-m.done
}
