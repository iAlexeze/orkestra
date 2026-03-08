package manager

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/health"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
)

type Manager struct {
	components []domain.Component
	postStart  []func(context.Context)
	timeout    time.Duration
	done       chan struct{}
	hs         *health.HealthServer
}

func NewManager(hs *health.HealthServer, timeout time.Duration) *Manager {
	return &Manager{
		timeout: timeout,
		hs:      hs,
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

	logger.Info().Msg("✅ All services started successfully")

	// Display started components
	fmt.Println("======================")
	fmt.Println("STARTED COMPONENTS:")
	n := 1
	for _, comp := range m.components {
		fmt.Printf("%d. %s\n", n, comp.Name())
		n++
	}
	fmt.Println("======================")

	// Run post-start hooks (leader election goes here)
	logger.Info().Msg("Running post-start hooks...")
	for _, hook := range m.postStart {
		go hook(mCtx)
	}

	m.setReady()
	logger.Info().Msg("controller is ready...")

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

		for _, comp := range m.components {
			name := comp.Name()
			logger.Info().Msgf("shutting down: %s...", name)
			if comp != nil {
				comp.Shutdown(shutdownCtx)
			}
			logger.Info().Msgf("%s status: %v", name, utils.StatusOffline)
		}

		logger.Info().Msg("✅ All services shut down gracefully")

		// Notify Wait() to terminate
		close(m.done)

	case <-ctx.Done():
		return
	}
}

// Register all components
func (m *Manager) Register(c []domain.Component) {
	fmt.Println("==================================")
	fmt.Println("REGISTERING MANAGER COMPONENTS...")
	for _, comp := range c {
		m.components = append(m.components, comp)
		logger.Info().Msgf("[%s] registered", comp.Name())
	}
	logger.Info().Msg("✅ All services registered successfully")

	// Display registered components
	fmt.Println("======================")
	fmt.Println("REGISTERED COMPONENTS:")
	n := 1
	for _, comp := range m.components {
		fmt.Printf("%d. %s\n", n, comp.Name())
		n++
	}
}

// AddPostStartHook: for services that need to start after manager has started
func (m *Manager) AddPostStartHook(hook func(context.Context)) {
	m.postStart = append(m.postStart, hook)
}

// setReady sets the controller ready after all startup is completed
func (m *Manager) setReady() {
	m.hs.SetReady()
}

// Listening to done channel
func (m *Manager) Wait() {
	<-m.done
}
