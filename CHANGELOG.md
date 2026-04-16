# CHANGELOG

## Enhancements

### Optional deletion‑protection and webhook cleanup  
Deletion protection and webhook cleanup are now fully optional and driven by explicit Katalog configuration.  
This change introduces a more flexible shutdown model:

- Deletion protection can be enabled or disabled independently of admission webhooks  
- Cleanup of validating/mutating/deletion‑protection webhooks is now controlled by the Katalog’s `deletionProtection.cleanupOnShutdown` flag  
- When cleanup is disabled, Orkestra leaves all webhook configurations intact across restarts  
- When cleanup is enabled, Orkestra removes only the webhook types that were declared in the Katalog

This ensures that operators can choose between **persistent** webhook configurations (production) and **ephemeral** cleanup (testing, CI, local development).

### Continuous webhook reconciliation  
A new background controller (`webhookController`) ensures that all declared webhooks remain available throughout the lifecycle of the Katalog.

This controller:

- Periodically verifies that validating, mutating, and deletion‑protection webhooks exist  
- Recreates missing webhook configurations  
- Ensures TLS and failure‑policy settings remain aligned with the Katalog  
- Operates independently of pod restarts, enabling long‑running consistency  
- Makes webhook availability **declarative and self‑healing**

This moves Orkestra toward a controller‑grade model where webhook configuration is continuously enforced rather than only applied at startup.

---

## Documentation updates (health.go)

The following architectural comments were added to `health.go`:

- Top‑level documentation for `HealthServer` describing its role as the runtime surface for health, admission, conversion, and deletion protection  
- Method‑level comments explaining lifecycle responsibilities (`Start`, `Shutdown`, `EnableWebhooks`, `EnableConversion`)  
- Inline comments at key architectural pivot points:
  - TLS validation for webhook servers  
  - Conditional registration of conversion, validation, mutation, and deletion‑protection endpoints  
  - Best‑effort webhook registration semantics  
  - Optional cleanup logic during shutdown  
  - Activation of the continuous webhook reconciliation loop  

These comments clarify the declarative model, the runtime activation path, and the separation between configuration, registration, and reconciliation.