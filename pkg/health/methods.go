package health

import (
	"github.com/orkspace/orkestra/pkg/katalog"
	"k8s.io/client-go/kubernetes"
)

// ── SetAdmissionRegistry ─────────────────────────────────────────────────────
//
// SetAdmissionRegistry provides the admission registry to the health server.
// Called from KomposeKatalogFromYaml after rules are loaded.
func (h *HealthServer) SetAdmissionRegistry(r katalog.AdmissionRegistry) {
	h.admissionRegistry = r
}

// SetKubeClient provides the Kubernetes client for webhook registration.
func (h *HealthServer) SetKubeClient(c kubernetes.Interface) {
	h.kubeClient = c
}

// SetWebhookOpts configures webhook registration options.
func (h *HealthServer) SetWebhookOpts(opts WebhookRegistrationOptions) {
	h.hookReg = opts
}

// SetCertManager provides the cert manager used to delete the TLS Secret on
// graceful shutdown when cleanupOnShutdown is enabled.
func (h *HealthServer) SetCertManager(m certManagerIface) {
	h.certMgr = m
}
