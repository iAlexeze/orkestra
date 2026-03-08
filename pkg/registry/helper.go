package registry

import (
	"reflect"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Update CRD Resource map for lookup
func updateResourceMapAndReturn(crds []crd) []crd {
	// Map the type of the object
	for _, c := range crds {
		resourceTypeMap[reflect.TypeOf(c.Object)] = c.Info.GroupVersionKind.String()
	}

	return crds
}

// Build CRDInfo
func cRDInfoFrom(group, version, kind, apiPath, plural, namespace string, clusterScoped bool) CRDInfo {
	if apiPath == "" {
		logger.Info().Msgf("API Path empty, using default: %s", DefaultAPIPath)
		apiPath = DefaultAPIPath
	}

	if group == "" && version == "" && kind == "" {
		logger.Error().Msg("required variables: Group, Version, Kind")
		panic("required variables: Group, Version, Kind")
	}

	if clusterScoped && namespace != "" {
		logger.Warn().Msgf("Resource %s/%s/%s is clusterscoped. Namespace %s will not be used.", group, version, kind, namespace)
		namespace = ""
	}

	return CRDInfo{
		Group:            group,
		Version:          version,
		Kind:             kind,
		GroupVersion:     &schema.GroupVersion{Group: group, Version: version},
		GroupVersionKind: schema.GroupVersionKind{Group: group, Version: version, Kind: kind},
		APIPath:          apiPath,
		NamePlural:       plural,
		Namespace:        namespace,
		ClusterScoped:    clusterScoped,
	}
}