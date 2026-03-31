package health

// Export internal functions for integration tests only.
var (
	ExportedApplyConversion  = applyConversion
	ExportRegisterWebhooks   = RegisterWebhooks
	ExportUnregisterWebhooks = UnregisterWebhooks

	ExportApplyValidatingWebhook = applyWebhookConfig
	ExportApplyMutatingWebhook   = applyMutatingWebhookConfig

	ExportCleanupValidatingWebhook = cleanupValidatingWebhook
	ExportCleanupMutatingWebhook   = cleanupMutatingWebhook
)
