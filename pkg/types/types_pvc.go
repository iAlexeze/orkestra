// pkg/types/types_pvc.go
package types

// PVCTemplateSource declares one PersistentVolumeClaim to be managed by Orkestra.
type PVCTemplateSource struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Name — PVC name. Required.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Namespace — target namespace. Default: CR namespace.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// StorageClassName — storage class to use. Empty means cluster default.
	StorageClassName string `yaml:"storageClassName,omitempty" json:"storageClassName,omitempty"`

	// AccessModes — access modes. Default: ["ReadWriteOnce"].
	// Supports: ReadWriteOnce, ReadOnlyMany, ReadWriteMany, ReadWriteOncePod.
	AccessModes []string `yaml:"accessModes,omitempty" json:"accessModes,omitempty"`

	// Storage — requested storage size (e.g. "10Gi"). Required.
	Storage string `yaml:"storage,omitempty" json:"storage,omitempty" validate:"required"`

	// VolumeMode — Filesystem or Block. Default: Filesystem.
	VolumeMode string `yaml:"volumeMode,omitempty" json:"volumeMode,omitempty"`

	// VolumeName — bind to a specific PV by name.
	VolumeName string `yaml:"volumeName,omitempty" json:"volumeName,omitempty"`

	Labels Labels `yaml:"labels,omitempty" json:"labels,omitempty"`

	Reconcile  bool         `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}

// PVTemplateSource declares one PersistentVolume to be managed by Orkestra.
// PersistentVolumes are cluster-scoped — Namespace is ignored.
type PVTemplateSource struct {
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Name — PV name. Required.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// StorageClassName — storage class name.
	StorageClassName string `yaml:"storageClassName,omitempty" json:"storageClassName,omitempty"`

	// Capacity — storage capacity (e.g. "10Gi"). Required.
	Capacity string `yaml:"capacity,omitempty" json:"capacity,omitempty" validate:"required"`

	// AccessModes — access modes. Default: ["ReadWriteOnce"].
	AccessModes []string `yaml:"accessModes,omitempty" json:"accessModes,omitempty"`

	// ReclaimPolicy — Retain, Delete, or Recycle. Default: Retain.
	ReclaimPolicy string `yaml:"reclaimPolicy,omitempty" json:"reclaimPolicy,omitempty"`

	// HostPath — host path for HostPath volume type. Used for local/dev PVs.
	HostPath string `yaml:"hostPath,omitempty" json:"hostPath,omitempty"`

	// CSI driver fields for cloud/CSI volumes.
	CSIDriver       string `yaml:"csiDriver,omitempty" json:"csiDriver,omitempty"`
	CSIVolumeHandle string `yaml:"csiVolumeHandle,omitempty" json:"csiVolumeHandle,omitempty"`

	Labels Labels `yaml:"labels,omitempty" json:"labels,omitempty"`

	Reconcile  bool         `yaml:"reconcile,omitempty" json:"reconcile,omitempty"`
	Conditions []Condition  `yaml:"when,omitempty" json:"when,omitempty"`
	AnyOf      []Condition  `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`
	ForEach    *ForEachSpec `yaml:"forEach,omitempty" json:"forEach,omitempty"`

	// Sleep injects an artificial delay into the reconcile of this resource.
	// Useful for autoscale testing, latency simulation, and chaos engineering.
	// Accepts extended duration units (s, m, h, d, w, mo, y).
	Sleep string `json:"sleep,omitempty" yaml:"sleep,omitempty"`
}
