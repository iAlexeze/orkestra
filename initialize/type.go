package initialize

import (
	"time"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/reconciler"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var ObjectRegistry = map[schema.GroupVersionKind]func() runtime.Object{}
var ListRegistry = map[schema.GroupVersionKind]func() runtime.Object{}

type CRDEntry struct {
	Name               string                        `yaml:"name" validate:"required,hostname_rfc1123"`
	ObjectGoMode       runtime.Object                `yaml:"-"`
	ListObjectGoMode   runtime.Object                `yaml:"-"`
	ObjectYamlMode     func() runtime.Object         `yaml:"-"`
	ListObjectYamlMode func() runtime.Object         `yaml:"-"`
	Reconciler         reconciler.NewReconcilerFunc  `yaml:"-"`
	Scheme             func(s *runtime.Scheme) error `yaml:"-"`
	Group              string                        `yaml:"group" validate:"required,hostname_rfc1123"`
	Version            string                        `yaml:"version" validate:"required"`
	Kind               string                        `yaml:"kind" validate:"required"`
	GroupVersion       *schema.GroupVersion          `yaml:"groupVersion" validate:"omitempty"`     // Optional (can be used if Group and Version are not specified)
	GroupVersionKind   schema.GroupVersionKind       `yaml:"groupVersionKind" validate:"omitempty"` //	Useful for some manipulations and Required by Registry
	NamePlural         string                        `yaml:"plural" validate:"required"`
	APIPath            string                        `yaml:"apiPath" validate:"omitempty"`
	Package            string                        `yaml:"package" validate:"required"` // Example: 'platform.ialexeze.io/v1alpha1'. Needed to construct scheme
	Namespaced         bool                          `yaml:"namespaced"`
	Namespace          string                        `yaml:"namespace"`
	Finalizer          string                        `yaml:"finalizer" validate:"omitempty"`
	LabelSelector      string                        `yaml:"labelSelector" validate:"omitempty"`
	ResyncPeriod       string                        `yaml:"resyncPeriod" validate:"omitempty"`
	Workers            int                           `yaml:"workers" validate:"omitempty,gte=1,lte=5"`
	Resync             time.Duration                 `yaml:"resync" validate:"omitempty"`
	DependsOn          []string                      `yaml:"dependsOn"`
}
