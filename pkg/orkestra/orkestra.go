package orkestra

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
)

const eventHandler = "event handler"

type Orkestra struct {
	komponents []domain.Komponent
	postStart  []postStart
	timeout    time.Duration
	logLevel   string
	done       chan struct{}
}

type postStart struct {
	hook func(context.Context)
	comp domain.Komponent
}

func NewOrkestra(timeout time.Duration, logLevel string) *Orkestra {
	return &Orkestra{
		timeout: timeout,
		done:    make(chan struct{}),
	}
}

func (o *Orkestra) Start(ctx context.Context) error {
	mCtx, mCancel := context.WithCancel(ctx)
	defer mCancel()

	logger.Info().Msg("Starting orkestra...")
	for _, comp := range o.komponents {
		name := comp.Name()

		logger.Debug().Msgf("[%s] starting...", name)
		if err := comp.Start(mCtx); err != nil {
			logger.Error().Err(err).Msgf("failed to start: %s", name)
			return err
		}
		utils.Sleep(1)
		logger.Debug().Msgf("%s status: %v", name, utils.StatusOnline)
	}

	// Run post-start hooks (leader election goes here)
	logger.Info().Msg("Running post-start hooks...")
	for _, p := range o.postStart {
		go p.hook(mCtx)
	}

	logger.Info().Msg("✅ All komponents started successfully")

	if strings.ToLower(o.logLevel) == "debug" {
		// Display started komponents
		fmt.Println("===============================")
		fmt.Println("STARTED KOMPONENTS:")

		n := 1
		for _, comp := range o.komponents {
			fmt.Printf("%d. %s\n", n, comp.Name())
			n++
		}

		for _, p := range o.postStart {
			fmt.Printf("%d. %s\n", n, p.comp.Name())
			n++
		}
		fmt.Println("===============================")

	}

	o.gracefulShutdown(mCtx, mCancel)
	return nil
}

func (o *Orkestra) Shutdown(ctx context.Context) {}

func (o *Orkestra) gracefulShutdown(ctx context.Context, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Warn().Msgf("recieved shutdown signal: %v", sig)
		logger.Warn().Msg("Shutting down orkestra...")
		cancel()

		// shutdown komponents
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), o.timeout)
		defer shutdownCancel()

		for _, comp := range utils.Reversed(o.komponents) {
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
			logger.Warn().Msgf("%s status: %v", name, utils.StatusOffline)
		}

		ev := o.GetKomponent(eventHandler)
		if ev != nil {
			ev.Shutdown(shutdownCtx)
			logger.Warn().Msgf("%s status: %v", ev.Name(), utils.StatusOffline)
		}

		logger.Warn().Msg("✅ All komponents shut down gracefully")

		// Notify Wait() to terminate
		close(o.done)

	case <-ctx.Done():
		return
	}
}

// Register all komponents
func (o *Orkestra) Register(c []domain.Komponent) {
	logger.Info().Msg("Registering orkestra komponents...")
	for _, comp := range c {
		o.komponents = append(o.komponents, comp)
		logger.Info().Msgf("[%s] registered", comp.Name())
	}
	logger.Info().Msg("✅ All komponents registered successfully")

	if strings.ToLower(o.logLevel) == "debug" {
		// Display registered komponents
		fmt.Println("==================================")
		fmt.Println("REGISTERED KOMPONENTS:")
		n := 1
		for _, comp := range o.komponents {
			fmt.Printf("%d. %s\n", n, comp.Name())
			n++
		}
	}
}

// GetKomponent returns a komponent if present
func (o *Orkestra) GetKomponent(name string) domain.Komponent {
	for _, comp := range o.komponents {
		if comp.Name() == name {
			return comp
		}
	}
	return nil
}

// AddPostStartHook: for services that need to start after Orkestra has started
func (o *Orkestra) AddPostStartHook(comp domain.Komponent, hook func(context.Context)) {
	o.postStart = append(o.postStart, postStart{
		hook: hook,
		comp: comp,
	})
}

// Listening to done channel
func (o *Orkestra) Wait() {
	<-o.done
}
