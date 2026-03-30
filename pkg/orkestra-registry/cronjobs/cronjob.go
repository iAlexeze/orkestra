// pkg/orkestra-registry/cronjobs/cronjob.go
package cronjobs

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResolvedCronJobSpec is the fully resolved CronJob specification.
type ResolvedCronJobSpec struct {
	// Name — CronJob name. Required.
	Name string

	// Namespace — target namespace. Required.
	Namespace string

	// Schedule — cron schedule expression. Required.
	// e.g. "0 * * * *" (every hour), "*/5 * * * *" (every 5 minutes)
	Schedule string

	// Image — container image. Required.
	Image string

	// Command — container entrypoint. Optional.
	Command []string

	// Args — container arguments. Optional.
	Args []string

	// Labels — applied to CronJob metadata.
	Labels map[string]string
}

// Create creates a CronJob if it does not already exist.
// Idempotent — skips if the CronJob exists.
// Owner reference set so CronJob is garbage collected when CR is deleted.
func Create(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedCronJobSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("cronjob.Create: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

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
		return fmt.Errorf("cronjob.Create: creating cronjob %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("cronjob", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("cronjob created")

	return nil
}

// Update reconciles an existing CronJob to match the resolved spec.
// Drift-corrects the schedule and image. If not found, creates it.
func Update(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedCronJobSpec) error {
	if err := validateSpec(spec); err != nil {
		return fmt.Errorf("cronjob.Update: invalid spec: %w", err)
	}

	namespace := resolveNamespace(owner, spec)

	existing, err := kube.Clientset().BatchV1().CronJobs(namespace).Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info().
				Str("cronjob", spec.Name).
				Str("namespace", namespace).
				Msg("cronjob not found during reconcile — recreating")
			return Create(ctx, kube, owner, spec)
		}
		return fmt.Errorf("cronjob.Update: getting cronjob %q: %w", spec.Name, err)
	}

	// Check for schedule or image drift
	currentSchedule := existing.Spec.Schedule
	currentImage := ""
	if len(existing.Spec.JobTemplate.Spec.Template.Spec.Containers) > 0 {
		currentImage = existing.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
	}

	if currentSchedule == spec.Schedule && currentImage == spec.Image {
		logger.Debug().
			Str("cronjob", spec.Name).
			Str("namespace", namespace).
			Msg("cronjob in sync — no update needed")
		return nil
	}

	updated := existing.DeepCopy()
	updated.Spec.Schedule = spec.Schedule
	if len(updated.Spec.JobTemplate.Spec.Template.Spec.Containers) > 0 {
		updated.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image = spec.Image
		updated.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command = spec.Command
		updated.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Args = spec.Args
	}

	_, err = kube.Clientset().BatchV1().CronJobs(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("cronjob.Update: updating cronjob %q: %w", spec.Name, err)
	}

	logger.Info().
		Str("cronjob", spec.Name).
		Str("namespace", namespace).
		Msg("cronjob updated")

	return nil
}

// Delete deletes the CronJob if it exists.
func Delete(ctx context.Context, kube *kubeclient.Kubeclient, owner domain.Object, spec ResolvedCronJobSpec) error {
	namespace := resolveNamespace(owner, spec)

	err := kube.Clientset().BatchV1().CronJobs(namespace).Delete(ctx, spec.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug().
				Str("cronjob", spec.Name).
				Str("namespace", namespace).
				Msg("cronjob already deleted — skipping")
			return nil
		}
		return fmt.Errorf("cronjob.Delete: deleting cronjob %q in %q: %w", spec.Name, namespace, err)
	}

	logger.Info().
		Str("cronjob", spec.Name).
		Str("namespace", namespace).
		Str("owner", owner.GetName()).
		Msg("cronjob deleted")

	return nil
}

// DeleteIfOwned deletes the CronJob if it exists and is owned by the CR.
func DeleteIfOwned(ctx context.Context, kube *kubeclient.Kubeclient,
	owner domain.Object, name, namespace string) error {

	existing, err := kube.Clientset().BatchV1().CronJobs(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	// Only delete if we own it
	if existing.Labels[konfig.LabelOrkestraOwner] != owner.GetName() {
		return nil
	}
	return kube.Clientset().BatchV1().CronJobs(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
}

// Resolve builds a ResolvedCronJobSpec from a CronJobTemplateSource.
// Template expressions must already be evaluated by template.Resolver before calling.
func Resolve(src orktypes.CronJobTemplateSource, ownerName string) ResolvedCronJobSpec {
	spec := ResolvedCronJobSpec{
		Name:      src.Name,
		Namespace: src.Namespace,
		Schedule:  src.Schedule,
		Image:     src.Image,
		Command:   src.Command,
		Args:      src.Args,
		Labels:    make(map[string]string),
	}

	if spec.Name == "" {
		spec.Name = ownerName + "-cronjob"
	}

	for _, l := range src.Labels {
		spec.Labels[l.Key] = l.Value
	}

	spec.Labels[konfig.LabelManaged] = konfig.LabelManagedValue
	spec.Labels[konfig.LabelOrkestraOwner] = ownerName

	return spec
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildCronJob(owner domain.Object, spec ResolvedCronJobSpec, namespace string) *batchv1.CronJob {
	return &batchv1.CronJob{
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
			Schedule: spec.Schedule,
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
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:    spec.Name,
									Image:   spec.Image,
									Command: spec.Command,
									Args:    spec.Args,
								},
							},
						},
					},
				},
			},
		},
	}
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

func resolveNamespace(owner domain.Object, spec ResolvedCronJobSpec) string {
	if spec.Namespace != "" {
		return spec.Namespace
	}
	if owner.GetNamespace() != "" {
		return owner.GetNamespace()
	}
	return "default"
}
