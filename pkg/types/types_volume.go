package types

// ── Volume / VolumeMount ──────────────────────────────────────────────────────
//
// VolumeSource and VolumeMount declare pod volumes and container mounts in the
// same way as Kubernetes, but with template expression support in name fields
// so volume names can be derived from the CR at reconcile time.
//
// Supported volume types: configMap, secret, emptyDir, persistentVolumeClaim,
// hostPath. Fields that accept template expressions are marked with (template).
//
// Example:
//
//	volumes:
//	  - name: app-config                          # (template)
//	    configMap:
//	      name: "{{ .metadata.name }}-config"     # (template)
//
//	  - name: creds
//	    secret:
//	      name: "{{ .metadata.name }}-creds"      # (template)
//
//	  - name: tmp
//	    emptyDir: {}
//
//	  - name: data
//	    persistentVolumeClaim:
//	      claimName: "{{ .metadata.name }}-pvc"   # (template)
//
//	  - name: host-logs
//	    hostPath:
//	      path: /var/log
//
//	volumeMounts:
//	  - name: app-config
//	    mountPath: /etc/config
//	    subPath: app.json
//	    readOnly: true

// VolumeSource declares one volume to attach to the pod.
// Exactly one of the source fields should be set.
type VolumeSource struct {
	// Name — volume name, referenced by volumeMounts. Supports template expressions.
	Name string `yaml:"name" json:"name"`

	// ConfigMap — mount a ConfigMap as a volume.
	ConfigMap *ConfigMapVolumeSource `yaml:"configMap,omitempty" json:"configMap,omitempty"`

	// Secret — mount a Secret as a volume.
	Secret *SecretVolumeSource `yaml:"secret,omitempty" json:"secret,omitempty"`

	// EmptyDir — an ephemeral empty directory. Exists for the lifetime of the pod.
	// Set to `{}` to use default settings.
	EmptyDir *EmptyDirVolumeSource `yaml:"emptyDir,omitempty" json:"emptyDir,omitempty"`

	// PersistentVolumeClaim — mount an existing PVC by claim name.
	PersistentVolumeClaim *PVCVolumeSource `yaml:"persistentVolumeClaim,omitempty" json:"persistentVolumeClaim,omitempty"`

	// HostPath — mount a file or directory from the host node's filesystem.
	// Use with care — host mounts are a security risk in multi-tenant clusters.
	HostPath *HostPathVolumeSource `yaml:"hostPath,omitempty" json:"hostPath,omitempty"`
}

// ConfigMapVolumeSource mounts a ConfigMap as a volume.
type ConfigMapVolumeSource struct {
	// Name — ConfigMap name. Supports template expressions.
	//   name: "{{ .metadata.name }}-config"
	Name string `yaml:"name" json:"name"`
}

// SecretVolumeSource mounts a Secret as a volume.
type SecretVolumeSource struct {
	// Name — Secret name. Supports template expressions.
	//   name: "{{ .metadata.name }}-creds"
	Name string `yaml:"name" json:"name"`
}

// EmptyDirVolumeSource is an ephemeral empty directory. Use `{}` in YAML.
type EmptyDirVolumeSource struct{}

// PVCVolumeSource mounts an existing PersistentVolumeClaim.
type PVCVolumeSource struct {
	// ClaimName — name of the PVC. Supports template expressions.
	ClaimName string `yaml:"claimName" json:"claimName"`

	// ReadOnly — mount the PVC read-only. Default: false.
	ReadOnly bool `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`
}

// HostPathVolumeSource mounts a path from the host node.
type HostPathVolumeSource struct {
	// Path — path on the host.
	Path string `yaml:"path" json:"path"`

	// Type — optional host path type. Values: DirectoryOrCreate, Directory,
	// FileOrCreate, File, Socket, CharDevice, BlockDevice.
	// Default: no type check.
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
}

// VolumeMount declares how a volume is mounted into the container.
type VolumeMount struct {
	// Name — matches a volume name declared in volumes:. Supports template expressions.
	Name string `yaml:"name" json:"name"`

	// MountPath — path inside the container where the volume is mounted.
	MountPath string `yaml:"mountPath" json:"mountPath"`

	// SubPath — path within the volume to mount. Defaults to "" (volume root).
	// Useful for mounting individual files from a ConfigMap or Secret.
	SubPath string `yaml:"subPath,omitempty" json:"subPath,omitempty"`

	// ReadOnly — mount the volume read-only. Default: false.
	ReadOnly bool `yaml:"readOnly,omitempty" json:"readOnly,omitempty"`
}
