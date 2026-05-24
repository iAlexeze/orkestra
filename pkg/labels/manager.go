// Package labels provides a manager for adding standard Orkestra labels and annotations
// to any Kubernetes object that implements the domain.Object interface.
//
// The manager is stateless and does not import any Orkestra internal packages (katalog, etc.).
// All configuration is passed explicitly at creation time.
//
// Responsibilities:
//   - EnsureManagedLabels: adds "orkestra.orkspace.io/managed: true" label,
//     and "managed-by", "managed-since" annotations. For runtime, also adds finalizers.
//   - EnsureDeletionProtectionLabel: adds "orkestra.io/deletion-protection: true" label
//     when deletion protection is enabled globally.
//
// The manager does NOT perform any API calls; it only mutates the object in memory.
// The caller is responsible for persisting changes (e.g., via kube.PatchLabels).
package labels

import (
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/logger"
)

// Manager handles label and annotation mutations on domain objects.
// It is configuration-driven and does not depend on external packages.
type Manager struct {
	// standalone indicates that no runtime reconciler is present.
	// When true, the manager will NOT add finalizers (they require a controller).
	standalone bool

	// deletionProtectionEnabled controls whether the deletion‑protection label
	// is applied. This should reflect the global security setting from Katalog.
	deletionProtectionEnabled bool

	deletionProtectionLabel string
	managedLabel            string
	managedByAnnotation     string
	managedSinceAnnotation  string
	strictModeExemptLabel   string
}

// Config holds the configuration for the LabelManager.
type Config struct {
	// Standalone mode: no runtime reconciler exists. Finalizers will be skipped.
	Standalone bool

	// DeletionProtectionEnabled: if true, the deletion‑protection label is added.
	DeletionProtectionEnabled bool
}

// NewManager creates a LabelManager with the given configuration.
func NewManager(cfg Config) *Manager {
	mgr := &Manager{
		standalone:                cfg.Standalone,
		deletionProtectionEnabled: cfg.DeletionProtectionEnabled,
	}

	mgr.deletionProtectionLabel = DeletionProtectionLabel
	mgr.strictModeExemptLabel = StrictModeExemptKey
	mgr.managedLabel = ManagedKey
	mgr.managedByAnnotation = AnnotationManagedBy
	mgr.managedSinceAnnotation = AnnotationManagedSince
	return mgr
}

// EnsureManagedLabel adds the standard Orkestra managed label to the object
// if it is missing.
//
// Managed label: orkestra.workspace.io/managed: "true"
//
// The method modifies the object in place. It returns true if the label was added
// or already present with the correct value, false if no change was needed.
func (m *Manager) EnsureManagedLabel(obj domain.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	if v, ok := labels[ManagedKey]; ok && v == ManagedValue {
		return false
	}
	labels[ManagedKey] = ManagedValue
	obj.SetLabels(labels)
	return true
}

// EnsureManagedAnnotations adds the standard Orkestra management annotations
// to the object if they are missing.
//
// Managed‑by annotation: orkestra.workspace.io/managed-by: <operatorName>
// Managed‑since annotation: orkestra.workspace.io/managed-since: <UTC timestamp>
//
// The method modifies the object in place. It returns true if any annotation
// was added or updated, false if both already had non‑empty values.
func (m *Manager) EnsureManagedAnnotations(obj domain.Object, operatorName string) bool {
	changed := false
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	if v, ok := ann[AnnotationManagedBy]; !ok || v == "" {
		ann[AnnotationManagedBy] = operatorName
		changed = true
	}
	if v, ok := ann[AnnotationManagedSince]; !ok || v == "" {
		ann[AnnotationManagedSince] = time.Now().UTC().Format(time.RFC3339)
		changed = true
	}
	if changed {
		obj.SetAnnotations(ann)
	}
	return changed
}

// EnsureDeletionProtectionLabel adds the deletion‑protection label to the object
// if the manager is configured with deletionProtectionEnabled = true.
//
// The label added is: orkestra.io/deletion-protection: "true"
//
// It returns true if the label was added or already present, false if the
// feature is disabled (no action taken).
// EnsureDeletionProtectionLabel ensures the deletion‑protection label is present
// when shouldHave is true, and absent when shouldHave is false.
// Returns true if the labels were modified.
func (m *Manager) EnsureDeletionProtectionLabel(obj domain.Object, shouldHave bool) bool {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	desiredValue := ""
	if shouldHave {
		desiredValue = DeletionProtectionValue
	}

	currentValue := labels[DeletionProtectionLabel]
	if currentValue == desiredValue {
		return false
	}

	if desiredValue == "" {
		delete(labels, DeletionProtectionLabel)
	} else {
		labels[DeletionProtectionLabel] = desiredValue
	}
	obj.SetLabels(labels)
	return true
}

// EnsureStrictModeExemptLabel ensures the strict‑mode exemption label is present
// when strict mode is disabled for this resource, and absent when strict mode is enabled.
// Returns true if the labels were modified.
func (m *Manager) EnsureStrictModeExemptLabel(obj domain.Object, strictModeEnabled bool) bool {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	desiredValue := ""
	if !strictModeEnabled {
		desiredValue = StrictModeExemptValue
	}

	currentValue := labels[StrictModeExemptKey]
	if currentValue == desiredValue {
		logger.Debug().
			Str("resource", obj.GetName()).
			Bool("strictModeEnabled", strictModeEnabled).
			Str("current", currentValue).
			Str("desired", desiredValue).
			Msg("label: exemption label already correct")
		return false // no change needed
	}

	if desiredValue == "" {
		delete(labels, StrictModeExemptKey)
		logger.Debug().
			Str("resource", obj.GetName()).
			Msg("label: removing exemption label")
	} else {
		labels[StrictModeExemptKey] = desiredValue
		logger.Debug().
			Str("resource", obj.GetName()).
			Msg("label: adding exemption label")
	}
	obj.SetLabels(labels)
	return true
}

// Getters

func (m *Manager) IsStandalone() bool {
	return m.standalone
}

func (m *Manager) IsDeletionProtectionEnabled() bool {
	return m.deletionProtectionEnabled
}

func (m *Manager) GetDeletionProtectionLabel() string {
	return m.deletionProtectionLabel
}

func (m *Manager) GetStrictModeExemptLabel() string {
	return m.strictModeExemptLabel
}

func (m *Manager) GetManagedLabel() string {
	return m.managedLabel
}

func (m *Manager) GetManagedByAnnotation() string {
	return m.managedByAnnotation
}

func (m *Manager) GetManagedSinceAnnotation() string {
	return m.managedSinceAnnotation
}
