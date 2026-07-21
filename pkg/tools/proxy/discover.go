package proxy

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

const (
	labelKomponent = "orkestra.orkspace.io/komponent"

	KomponentRuntime = "runtime"
	KomponentCC      = "control-center"
	KomponentGateway = "gateway"

	runtimeLeaseName = orktypes.KonductorLeaseName
)

// FoundService is a discovered Orkestra service and its port.
type FoundService struct {
	Name string
	Port int32
}

// FindService lists Services with the given komponent label in ns.
// Returns nil, nil when no matching service is found (component not deployed).
func FindService(ctx context.Context, cs kubernetes.Interface, ns, komponent string) (*FoundService, error) {
	sel := labelKomponent + "=" + komponent
	list, err := cs.CoreV1().Services(ns).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, fmt.Errorf("list services (%s): %w", sel, err)
	}
	if len(list.Items) == 0 || len(list.Items[0].Spec.Ports) == 0 {
		return nil, nil
	}
	svc := &list.Items[0]
	return &FoundService{Name: svc.Name, Port: svc.Spec.Ports[0].Port}, nil
}

// ResolvePod returns the name of a running pod backing the given service.
func ResolvePod(ctx context.Context, cs kubernetes.Interface, ns, svcName string) (string, error) {
	svc, err := cs.CoreV1().Services(ns).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get service %s: %w", svcName, err)
	}
	if len(svc.Spec.Selector) == 0 {
		return "", fmt.Errorf("service %s has no selector", svcName)
	}

	var parts []string
	for k, v := range svc.Spec.Selector {
		parts = append(parts, k+"="+v)
	}
	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: strings.Join(parts, ","),
	})
	if err != nil {
		return "", fmt.Errorf("list pods for %s: %w", svcName, err)
	}
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning {
			return p.Name, nil
		}
	}
	return "", fmt.Errorf("no running pod for %s", svcName)
}

// ResolveRuntimePod reads the Runtime leader Lease and returns the holder pod name.
// Uses the Go client directly — not a kubectl subprocess.
func ResolveRuntimePod(ctx context.Context, cs kubernetes.Interface, ns string) (string, error) {
	lease, err := cs.CoordinationV1().Leases(ns).Get(ctx, runtimeLeaseName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get lease %s/%s: %w", ns, runtimeLeaseName, err)
	}
	if lease.Spec.HolderIdentity == nil || strings.TrimSpace(*lease.Spec.HolderIdentity) == "" {
		return "", fmt.Errorf("lease %s/%s has no holder yet", ns, runtimeLeaseName)
	}
	return strings.TrimSpace(*lease.Spec.HolderIdentity), nil
}
