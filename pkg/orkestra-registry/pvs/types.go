// pkg/orkestra-registry/pvs/types.go
package pvs

// ResolvedPVSpec is the fully resolved PersistentVolume specification.
// PersistentVolumes are cluster-scoped — Namespace is not used.
type ResolvedPVSpec struct {
	Name             string
	StorageClassName string
	Capacity         string
	AccessModes      []string
	ReclaimPolicy    string
	HostPath         string
	CSIDriver        string
	CSIVolumeHandle  string
	Labels           map[string]string
}
