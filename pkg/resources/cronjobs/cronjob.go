// pkg/resources/cronjobs/cronjob.go
package cronjobs

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	"github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/resources/common"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedCronJobSpec is the fully resolved CronJob specification.
// All template expressions have been evaluated before this struct is populated.
type ResolvedCronJobSpec struct {
	// Name — CronJob name. Required.
	Name string

	// Namespace — target namespace.
	Namespace string

	// Schedule — standard cron expression. Required.
	// e.g. "*/5 * * * *", "0 2 * * 1-5"
	Schedule string

	// Image — container image. Required.
	Image string

	// Command — container entrypoint override.
	Command []string

	// Args — container arguments.
	Args []string

	// Suspend — when true, all subsequent executions are suspended.
	// The currently running job is not affected.
	Suspend bool

	// ConcurrencyPolicy — how to treat concurrent executions.
	// One of: Allow (default), Forbid, Replace.
	ConcurrencyPolicy batchv1.ConcurrencyPolicy

	// StartingDeadlineSeconds — deadline for starting the job if it
	// misses its scheduled time. nil means no deadline.
	StartingDeadlineSeconds *int64

	// SuccessfulJobsHistoryLimit — number of successful finished jobs to keep.
	// nil means the Kubernetes default (3).
	SuccessfulJobsHistoryLimit *int32

	// FailedJobsHistoryLimit — number of failed finished jobs to keep.
	// nil means the Kubernetes default (1).
	FailedJobsHistoryLimit *int32

	// Labels — applied to CronJob and pod metadata.
	Labels map[string]string

	// Resources — CPU and memory requests/limits. nil means no limits set.
	Resources *orktypes.ResourceRequirements

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use
	// for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use.
	ImagePullSecrets []string

	// SecurityContext — container-level security settings.
	SecurityContext *orktypes.ContainerSecurityContext

	// PodSecurity — pod-level security settings.
	PodSecurity *orktypes.PodSecurityContext

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string
}

// Create creates a CronJob if it does not already exist.
// Idempotent — skips creation if the CronJob already exists.
// Sets owner reference so the CronJob is garbage collected when the CR is deleted.
func Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedCronJobSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("cronjob.Create: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	_, err := kube.Clientset().BatchV1().CronJobs(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("cronjob.Create: checking existence of %q: %w", spec.Name, err)
	}
	if err == nil {
		logger.Debug().
			Str("cronjob", spec.Name).
			Str("namespace", namespace).
			Msg("cronjob already exists — skipping create")
		return nil
	}

	cj := buildCronJob(owner, spec, namespace)

	_, err = kube.Clientset().BatchV1().CronJobs(namespace).Create(ctx, cj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("cronjob.Create: creating %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("cronjob", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("cronjob created")

	return nil
}

// Update reconciles an existing CronJob to match the resolved spec.
// Detects drift across schedule, image, suspend, concurrencyPolicy,
// and history limits. Creates the CronJob if it does not exist.
func Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedCronJobSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("cronjob.Update: %w", err)
	}

	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	existing, err := kube.Clientset().BatchV1().CronJobs(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("cronjob", spec.Name).
				Str("namespace", namespace).
				Msg("cronjob not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("cronjob.Update: getting %q: %w", spec.Name, err)
	}

	// ── Drift detection ───────────────────────────────────────────────────
	drifted := false
	updated := existing.DeepCopy()

	if existing.Spec.Schedule != spec.Schedule {
		updated.Spec.Schedule = spec.Schedule
		drifted = true
		logger.Info().Str("cronjob", spec.Name).
			Str("desired", spec.Schedule).Msg("cronjob schedule drifted")
	}

	if existing.Spec.Suspend == nil || *existing.Spec.Suspend != spec.Suspend {
		updated.Spec.Suspend = utils.BoolPtr(spec.Suspend)
		drifted = true
		logger.Info().Str("cronjob", spec.Name).
			Bool("desired", spec.Suspend).Msg("cronjob suspend drifted")
	}

	if spec.ConcurrencyPolicy != "" && existing.Spec.ConcurrencyPolicy != spec.ConcurrencyPolicy {
		updated.Spec.ConcurrencyPolicy = spec.ConcurrencyPolicy
		drifted = true
	}

	if spec.StartingDeadlineSeconds != nil {
		if existing.Spec.StartingDeadlineSeconds == nil ||
			*existing.Spec.StartingDeadlineSeconds != *spec.StartingDeadlineSeconds {
			updated.Spec.StartingDeadlineSeconds = spec.StartingDeadlineSeconds
			drifted = true
		}
	}

	if spec.SuccessfulJobsHistoryLimit != nil {
		if existing.Spec.SuccessfulJobsHistoryLimit == nil ||
			*existing.Spec.SuccessfulJobsHistoryLimit != *spec.SuccessfulJobsHistoryLimit {
			updated.Spec.SuccessfulJobsHistoryLimit = spec.SuccessfulJobsHistoryLimit
			drifted = true
		}
	}

	if spec.FailedJobsHistoryLimit != nil {
		if existing.Spec.FailedJobsHistoryLimit == nil ||
			*existing.Spec.FailedJobsHistoryLimit != *spec.FailedJobsHistoryLimit {
			updated.Spec.FailedJobsHistoryLimit = spec.FailedJobsHistoryLimit
			drifted = true
		}
	}

	if len(existing.Spec.JobTemplate.Spec.Template.Spec.Containers) > 0 {
		container := &updated.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
		if container.Image != spec.Image {
			container.Image = spec.Image
			drifted = true
			logger.Info().Str("cronjob", spec.Name).
				Str("desired", spec.Image).Msg("cronjob image drifted")
		}
		if len(spec.Command) > 0 && !reflect.DeepEqual(container.Command, spec.Command) {
			container.Command = spec.Command
			drifted = true
		}
		if len(spec.Args) > 0 && !reflect.DeepEqual(container.Args, spec.Args) {
			container.Args = spec.Args
			drifted = true
		}
		if spec.Resources != nil {
			desiredRes := common.BuildResourceRequirements(spec.Resources)
			if !common.ResourceRequirementsEqual(container.Resources, desiredRes) {
				container.Resources = desiredRes
				drifted = true
				logger.Info().Str("cronjob", spec.Name).Msg("cronjob resources drifted")
			}
		}
	}

	if !drifted {
		logger.Debug().
			Str("cronjob", spec.Name).
			Str("namespace", namespace).
			Msg("cronjob in sync — no update needed")
		return nil
	}

	_, err = kube.Clientset().BatchV1().CronJobs(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("cronjob.Update: updating %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("cronjob", spec.Name).
		Str("namespace", namespace).
		Msg("cronjob updated")

	return nil
}

// Delete deletes the CronJob if it exists.
func Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedCronJobSpec) error {
	namespace := common.ResolveNamespace(owner, spec.Namespace)
	if err := common.SleepIfNeeded(spec.Sleep); err != nil {
		return err
	}

	err := kube.Clientset().BatchV1().CronJobs(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("cronjob", spec.Name).
				Str("namespace", namespace).
				Msg("cronjob already deleted — skipping")
			return nil
		}
		return fmt.Errorf("cronjob.Delete: deleting %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("cronjob", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("cronjob deleted")

	return nil
}

// DeleteIfOwned deletes the CronJob only if it is labelled as owned by the CR.
func DeleteIfOwned(ctx context.Context, kube kubeclient.KubeClient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().BatchV1().CronJobs(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cronjob.DeleteIfOwned: getting %q: %w", name, err)
	}
	if existing.Labels[labels.OrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().BatchV1().CronJobs(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedCronJobSpec from a CronJobTemplateSource.
// All template expressions in src must already have been evaluated by
// template.Resolver — Resolve only performs type conversion and defaults.
func Resolve(src orktypes.CronJobTemplateSource, ownerName string, reg orktypes.ProfileRegistry) ResolvedCronJobSpec {
	spec := ResolvedCronJobSpec{
		Name:            src.Name,
		Namespace:       src.Namespace,
		Schedule:        src.Schedule,
		Image:           src.Image,
		Command:         src.Command,
		Args:            src.Args,
		Labels:          make(map[string]string),
		Resources:       common.ResolveResources(src.Resources, reg),
		SecurityContext: common.ResolveContainerSecurityContext(src.SecurityContext, reg),
		PodSecurity:     common.ResolvePodSecurityContext(src.PodSecurity, reg),
		Sleep:           src.Sleep,
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-cronjob"
	}

	// ── Suspend ───────────────────────────────────────────────────────────
	if src.Suspend != "" {
		spec.Suspend = common.ParseBool(src.Suspend)
	}

	// ── ConcurrencyPolicy ─────────────────────────────────────────────────
	switch strings.ToLower(src.ConcurrencyPolicy) {
	case "forbid":
		spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
	case "replace":
		spec.ConcurrencyPolicy = batchv1.ReplaceConcurrent
	default:
		spec.ConcurrencyPolicy = batchv1.AllowConcurrent
	}

	// ── StartingDeadlineSeconds ───────────────────────────────────────────
	if src.StartingDeadlineSeconds != "" {
		if n, err := strconv.ParseInt(src.StartingDeadlineSeconds, 10, 64); err == nil && n > 0 {
			spec.StartingDeadlineSeconds = &n
		}
	}

	// ── SuccessfulJobsHistoryLimit ────────────────────────────────────────
	if src.SuccessfulJobsHistoryLimit != "" {
		if n, err := strconv.ParseInt(src.SuccessfulJobsHistoryLimit, 10, 32); err == nil {
			n32 := int32(n)
			spec.SuccessfulJobsHistoryLimit = &n32
		}
	} else {
		n := int32(3) // Kubernetes default
		spec.SuccessfulJobsHistoryLimit = &n
	}

	// ── FailedJobsHistoryLimit ────────────────────────────────────────────
	if src.FailedJobsHistoryLimit != "" {
		if n, err := strconv.ParseInt(src.FailedJobsHistoryLimit, 10, 32); err == nil {
			n32 := int32(n)
			spec.FailedJobsHistoryLimit = &n32
		}
	} else {
		n := int32(1) // Kubernetes default
		spec.FailedJobsHistoryLimit = &n
	}

	// ── Labels ────────────────────────────────────────────────────────────
	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	spec.Labels[labels.ManagedKey] = labels.ManagedValue
	spec.Labels[labels.OrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildCronJob(owner domain.Object, spec ResolvedCronJobSpec, namespace string) *batchv1.CronJob {
	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: namespace,
			Labels:    spec.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         owner.GetObjectKind().GroupVersionKind().GroupVersion().String(),
					Kind:               owner.GetObjectKind().GroupVersionKind().Kind,
					Name:               owner.GetName(),
					UID:                owner.GetUID(),
					Controller:         utils.BoolPtr(true),
					BlockOwnerDeletion: utils.BoolPtr(true),
				},
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   spec.Schedule,
			Suspend:                    utils.BoolPtr(spec.Suspend),
			ConcurrencyPolicy:          spec.ConcurrencyPolicy,
			StartingDeadlineSeconds:    spec.StartingDeadlineSeconds,
			SuccessfulJobsHistoryLimit: spec.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     spec.FailedJobsHistoryLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: spec.Labels,
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: spec.Labels,
						},
						Spec: corev1.PodSpec{
							ImagePullSecrets: common.ToPullSecrets(spec.ImagePullSecrets),
							RestartPolicy:    corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								buildContainer(spec),
							},
						},
					},
				},
			},
		},
	}

	// Security
	common.ApplySecurityContext(
		&cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0],
		&cj.Spec.JobTemplate.Spec.Template.Spec,
		spec.SecurityContext,
		spec.PodSecurity,
	)

	return cj
}

func buildContainer(spec ResolvedCronJobSpec) corev1.Container {
	c := corev1.Container{
		Name:    spec.Name,
		Image:   spec.Image,
		Command: spec.Command,
		Args:    spec.Args,
	}
	if spec.Resources != nil {
		c.Resources = common.BuildResourceRequirements(spec.Resources)
	}
	return c
}

func validateSpec(spec ResolvedCronJobSpec) error {
	var missing []string
	if spec.Name == "" {
		missing = append(missing, "name")
	}
	if spec.Image == "" {
		missing = append(missing, "image")
	}
	if spec.Schedule == "" {
		missing = append(missing, "schedule")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}
