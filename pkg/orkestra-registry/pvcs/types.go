// pkg/orkestra-registry/pvcs/types.go
package pvcs

// ResolvedPVCSpec is the fully resolved PersistentVolumeClaim specification.
type ResolvedPVCSpec struct {
	Name             string
	Namespace        string
	StorageClassName string
	AccessModes      []string
	Storage          string
	VolumeMode       string
	VolumeName       string
	Labels           map[string]string
}
