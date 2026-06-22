//go:build ignore

package controller

import (
	// TODO(ork migrate): add these imports:
	//   "github.com/orkspace/orkestra/domain"
	//   "github.com/orkspace/orkestra/pkg/event"
	//   "github.com/orkspace/orkestra/pkg/kubeclient"
	//   "k8s.io/client-go/tools/cache"
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	demov1alpha1 "github.com/example/webapp-operator/api/v1alpha1"
)

// WebAppReconciler reconciles a WebApp object.
type WebAppReconciler struct {
	informer cache.SharedIndexInformer
	kube     kubeclient.KubeClient
	ev       event.Recorder
}

// +kubebuilder:rbac:groups=migration.demo.orkestra.io,resources=webapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=migration.demo.orkestra.io,resources=webapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *WebAppReconciler) Reconcile(ctx context.Context, key string) error {
	parts := strings.SplitN(key, "/", 2)
	namespace, name := parts[0], parts[1]

	logger := log.FromContext(ctx)

	// Fetch the WebApp CR
	webapp := &demov1alpha1.WebApp{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, webapp); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, webapp); err != nil {
		logger.Error(err, "failed to reconcile Deployment")
		return err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, webapp); err != nil {
		logger.Error(err, "failed to reconcile Service")
		return err
	}

	// Update status
	webapp.Status.Phase = "Running"
	webapp.Status.Endpoint = fmt.Sprintf("%s.%s.svc.cluster.local", webapp.Name, webapp.Namespace)
	webapp.Status.Replicas = webapp.Spec.Replicas
	if err := nil; /* TODO(ork migrate): replace with r.kube.PatchStatus(ctx, obj, GroupVersionResource, map[string]interface{}{...}) */ err != nil {
		logger.Error(err, "failed to update WebApp status")
		return err
	}

	return nil
}

func (r *WebAppReconciler) reconcileDeployment(ctx context.Context, webapp *demov1alpha1.WebApp) error {
	replicas := webapp.Spec.Replicas
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name,
			Namespace: webapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(webapp, demov1alpha1.GroupVersionKind),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": webapp.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": webapp.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  webapp.Name,
							Image: webapp.Spec.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: webapp.Spec.Port},
							},
						},
					},
				},
			},
		},
	}

	existing := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Name: webapp.Name, Namespace: webapp.Namespace}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, deploy)
	}
	if err != nil {
		return err
	}
	patch := client.MergeFrom(existing.DeepCopy())
	existing.Spec = deploy.Spec
	return r.Patch(ctx, existing, patch)
}

func (r *WebAppReconciler) reconcileService(ctx context.Context, webapp *demov1alpha1.WebApp) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name + "-svc",
			Namespace: webapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(webapp, demov1alpha1.GroupVersionKind),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": webapp.Name},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(int(webapp.Spec.Port)),
				},
			},
		},
	}

	existing := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Name: webapp.Name + "-svc", Namespace: webapp.Namespace}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, svc)
	}
	if err != nil {
		return err
	}
	patch := client.MergeFrom(existing.DeepCopy())
	existing.Spec.Ports = svc.Spec.Ports
	return r.Patch(ctx, existing, patch)
}

// SetupWithManager removed — Orkestra provides the informer, workqueue,
// worker pool, leader election, panic recovery, and metrics.
// Delete this file's main.go and scheme registration too.

// NewWebAppReconciler is the constructor function registered in the Katalog.
func NewWebAppReconciler(
	kube kubeclient.KubeClient,
	informer cache.SharedIndexInformer,
	ev event.Recorder,
) domain.Reconciler {
	return &WebAppReconciler{
		kube:     kube,
		informer: informer,
		ev:       ev,
	}
}
