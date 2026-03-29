// health/server_update.go
//
// This file shows the additions to health/health.go needed for admission support.
// Add these fields, options, and startup logic to the existing HealthServer.

package health

// ── Additions to HealthServer struct ─────────────────────────────────────────
//
// Add these fields to the existing HealthServer struct in health.go:
//
//   // admissionRegistry holds validation and mutation rules loaded from the Katalog.
//   // Populated at startup from the Katalog when ENABLE_WEBHOOKS=true.
//   admissionRegistry katalog.AdmissionRegistry
//
//   // webhookOpts holds the configuration for webhook registration.
//   webhookOpts WebhookRegistrationOptions
//
//   // kubeClient is used for webhook configuration registration.
//   // Set via SetKubeClient after the HealthServer is constructed.
//   kubeClient kubernetes.Interface

// ── Additions to WebhookConfgurationOptions ───────────────────────────────────────────────
//
// Add ENABLE_WEBHOOKS support alongside ENABLE_CONVERSION.
// In Start(), add:
//

// ── Route registration ────────────────────────────────────────────────────────
//
// In Start(), after registering /convert (when ENABLE_CONVERSION=true),
// add the following block for ENABLE_WEBHOOKS=true:
//

// ── Options additions ───────────────────────────────────────────────
//
// Add WebhooksEnabled to the existing Options struct:
//
//   type Options struct {
//       ConvEnabled     bool
//       ConvCert        string
//       ConvKey         string
//       WebhooksEnabled bool   // ← new: enable /validate and /mutate endpoints
//   }
//
// Read from environment in konstructOrkestra:
//   WebhooksEnabled: os.Getenv("ENABLE_WEBHOOKS") == "true"

// ── Environment variables ─────────────────────────────────────────────────────
//
// ENABLE_CONVERSION=true  → /convert endpoint (existing)
// ENABLE_WEBHOOKS=true    → /validate and /mutate endpoints (new)
//                           requires ENABLE_CONVERSION=true (shares the HTTPS server)
//
// All three endpoints share:
//   - The same HTTPS server on :8443
//   - The same TLS_CERT and TLS_KEY
//   - The same CA bundle used in webhook configurations
//
// This is architecturally clean: one server, one certificate, all admission
// and conversion concerns handled by the same trusted endpoint.
