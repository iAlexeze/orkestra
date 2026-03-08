package leader

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type leaderElection struct {
	name       string
	kube       *kubeclient.Kubeclient
	event      *event.Event
	cancelFunc context.CancelFunc
	run        func(context.Context)

	election theElection
	opts     Options
}

type theElection struct {
	startedLeading atomic.Bool
	leader         string
}

type Options struct {
	LeaseDuration time.Duration
	RetryPeriod   time.Duration
	RenewDeadline time.Duration

	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
}

var _ domain.Component = (*leaderElection)(nil)

func NewLeaderElection(
	kube *kubeclient.Kubeclient,
	event *event.Event,
	run func(context.Context),
	opts Options,
) *leaderElection {
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}

	le := &leaderElection{
		name:  "resource-leader",
		event: event,
		kube:  kube,
		run:   run,
		opts:  opts,
	}

	le.election.startedLeading.Store(false)
	le.election.leader = ""

	return le
}

func (le *leaderElection) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create a cancellable context for the leader election
	leaderCtx, cancel := context.WithCancel(ctx)
	le.cancelFunc = cancel

	go func() {
		leaderelection.RunOrDie(leaderCtx, le.leaseConfig())
	}()
	return nil
}

func (le *leaderElection) Shutdown(ctx context.Context) {
	logger.Info().Msg("🛑 Shutting down leader election...")

	// Cancel the leader election context
	if le.cancelFunc != nil {
		le.cancelFunc()
	}

	// Give it a moment to release the lease
	utils.Sleep(2)
	logger.Info().Msg("✅ Leader election shut down")
}

func (le *leaderElection) Name() string {
	return le.name
}

func (le *leaderElection) kind() string {
	return "Lease"
}

// Helpers
// Lease configuration
func (le *leaderElection) leaseConfig() leaderelection.LeaderElectionConfig {
	return leaderelection.LeaderElectionConfig{
		Name:            le.Name(),
		Lock:            le.leaseLock(),
		LeaseDuration:   le.opts.LeaseDuration,
		RenewDeadline:   le.opts.RenewDeadline,
		RetryPeriod:     le.opts.RetryPeriod,
		ReleaseOnCancel: true,
		Callbacks:       le.callbacks(),
	}
}

// Lease lock
func (le *leaderElection) leaseLock() *resourcelock.LeaseLock {
	opts := le.opts
	return &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:        le.name,
			Namespace:   opts.Namespace,
			Annotations: opts.Annotations,
			Labels:      opts.Labels,
		},
		Client: le.kube.Clientset().CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      hostname(),
			EventRecorder: le.event.Recorder(),
		},
	}
}

// Build callbacks
func (le *leaderElection) callbacks() leaderelection.LeaderCallbacks {
	return leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			if le.event.Recorder() != nil {
				le.event.Recorder().Eventf(
					&corev1.ObjectReference{
						Name:      le.name,
						Namespace: le.opts.Namespace,
						Kind:      le.kind(),
					}, corev1.EventTypeNormal, "LeaderElected", "%s became leader", hostname(),
				)
			}

			le.election.leader = hostname()
			logger.Info().Msgf("%s 🏆 became leader, starting controller...", le.election.leader)
			le.election.startedLeading.Store(true)

			le.run(ctx)
		},
		OnStoppedLeading: func() {
			if le.event.Recorder() != nil {
				le.event.Recorder().Eventf(
					&corev1.ObjectReference{
						Name:      le.name,
						Namespace: le.opts.Namespace,
						Kind:      le.kind(),
					}, corev1.EventTypeWarning, "LeaderLost", "%s lost leadership", hostname(),
				)
			}
			// Can perform additional clean up if needed on the actual leader
			// Example
			if le.election.startedLeading.Load() {
				logger.Info().Msg("Performing cleanup on the actual leader...")
				le.election.leader = ""
				le.election.startedLeading.Store(false)
			} else {
				logger.Info().Msg("No cleanup needed as we never started leading.")
			}

			logger.Info().Msgf("%s👋 Stopped leading - lease released", hostname())
		},
		OnNewLeader: func(identity string) {
			if le.event.Recorder() != nil {
				le.event.Recorder().Eventf(
					&corev1.ObjectReference{
						Name:      le.name,
						Namespace: le.opts.Namespace,
						Kind:      le.kind(),
					}, corev1.EventTypeNormal, "NewLeaderElected", "%s elected as leader", hostname(),
				)
			}
			logger.Info().Msgf("👑 New leader elected: %s", identity)
		},
	}
}

// Get hostname
func hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = uuid.New().String()
	}
	return hostname
}

// Leader returns the instance that won the leader election
func (le *leaderElection) Leader() string {
	return le.election.leader
}
