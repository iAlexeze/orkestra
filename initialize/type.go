package initialize

import (
	"time"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/reconciler"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var ObjectRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var ListRegistry = map[schema.GroupVersionKind]func() runtime.Object{}

type Queue struct {
	Default          bool `yaml:"default"`
	MaxQueueDepth    int  `yaml:"maxQueueDepth"`
	DegradeThreshold int  `yaml:"degradeThreshold"`
}

type ReconcilerConfig struct {
	// Default: true → use GenericReconciler with whatever HookFactory returns
	Default bool `yaml:"default" validate:"omitempty"`

	// Custom constructor — used when Default: false
	// signature matches what buildManager already calls
	Constructor reconciler.NewReconcilerFunc

	// HookFactory — called by buildManager to get hooks for GenericReconciler
	// Only used when Default: true
	// If nil AND Default: true → empty hooks (pure watch/list)
	HookFactory func() domain.AnyReconcileHooks

	// Finalizers - per CRD finalizer. overwrites the one set on the Katalog resource
	Finalizers []string `yaml:"finalizers" validate:"omitempty"`
}

type CRDEntry struct {
	Name                 string                        `yaml:"name" validate:"required,hostname_rfc1123"`
	Enabled              bool                          `yaml:"enabled"`
	Critical             bool                          `yaml:"critical"`
	Description          string                        `yaml:"description"`
	ObjectGoMode         runtime.Object                `yaml:"-"`
	ListObjectGoMode     runtime.Object                `yaml:"-"`
	ObjectYamlMode       func() runtime.Object         `yaml:"-"`
	ListObjectYamlMode   func() runtime.Object         `yaml:"-"`
	ReconcilerConfig     ReconcilerConfig              `yaml:"reconciler"`
	Queue                Queue                         `yaml:"queue"`
	Scheme               func(s *runtime.Scheme) error `yaml:"-"`
	Group                string                        `yaml:"group" validate:"required,hostname_rfc1123"`
	Version              string                        `yaml:"version" validate:"required"`
	Kind                 string                        `yaml:"kind" validate:"required"`
	GroupVersion         *schema.GroupVersion          `yaml:"groupVersion" validate:"omitempty"`     // Optional (can be used if Group and Version are not specified)
	GroupVersionKind     schema.GroupVersionKind       `yaml:"groupVersionKind" validate:"omitempty"` //	Useful for some manipulations and Required by Registry
	GroupVersionResource schema.GroupVersionResource   `yaml:"groupVersionResource" validate:"omitempty"`
	Plural               string                        `yaml:"plural" validate:"required"`
	APIPath              string                        `yaml:"apiPath" validate:"omitempty"`
	Package              string                        `yaml:"package" validate:"required"` // Example: 'platform.orkestra.io/v1alpha1'. Needed to construct scheme
	Namespaced           bool                          `yaml:"namespaced"`
	Namespace            string                        `yaml:"namespace"`
	LabelSelector        string                        `yaml:"labelSelector" validate:"omitempty"`
	ResyncPeriod         string                        `yaml:"resyncPeriod" validate:"omitempty"`
	Workers              int                           `yaml:"workers" validate:"omitempty,gte=1,lte=5"`
	Resync               time.Duration                 `yaml:"resync" validate:"omitempty"`
	DependsOn            []string                      `yaml:"dependsOn"`
}

func (c *CRDEntry) GetRuntimeObjects(mode string) (runtime.Object, runtime.Object) {
	if mode == "yaml" {
		return c.ObjectYamlMode(), c.ListObjectYamlMode()
	}
	return c.ObjectGoMode, c.ListObjectGoMode
}

func (c *CRDEntry) SetMaxQueueDepth(def int) int {
	if c.Queue.MaxQueueDepth == 0 {
		return def
	}
	return c.Queue.MaxQueueDepth
}
