package health

// Export internal functions for integration tests only.
var (
	ExportedApplyConversion           = applyConversion
	ExportRegisterAdmissionWebhooks   = RegisterAdmissionWebhooks
	ExportUnregisterAdmissionWebhooks = UnregisterAdmissionWebhooks

	ExportApplyValidatingWebhook = applyWebhookConfig
	ExportApplyMutatingWebhook   = applyMutatingWebhookConfig

	ExportCleanupValidatingWebhook = cleanupValidatingWebhook
	ExportCleanupMutatingWebhook   = cleanupMutatingWebhook
)
