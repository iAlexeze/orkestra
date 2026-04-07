// pkg/inspect/discovery_test.go
package inspect_test

import (
	"testing"

	"github.com/ialexeze/orkestra/pkg/inspect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery/fake"
	faketesting "k8s.io/client-go/testing"
)

// fakeDiscovery builds a fake discovery client with the given API groups.
func fakeDiscovery(resources ...*metav1.APIResourceList) *fake.FakeDiscovery {
	disc := &fake.FakeDiscovery{
		Fake: &faketesting.Fake{},
	}
	disc.Resources = resources
	return disc
}

func apiResource(groupVersion, kind, plural, singular string, namespaced bool) *metav1.APIResourceList {
	return &metav1.APIResourceList{
		GroupVersion: groupVersion,
		APIResources: []metav1.APIResource{
			{
				Name:         plural,
				SingularName: singular,
				Kind:         kind,
				Namespaced:   namespaced,
			},
		},
	}
}

func TestDiscoverCRD_ByPlural(t *testing.T) {
	disc := fakeDiscovery(
		apiResource("demo.orkestra.io/v1alpha1", "Website", "websites", "website", true),
	)

	crd, err := inspect.DiscoverCRD(disc, "websites")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if crd.Kind != "Website" {
		t.Errorf("expected Kind=Website, got %q", crd.Kind)
	}
	if crd.Group != "demo.orkestra.io" {
		t.Errorf("expected Group=demo.orkestra.io, got %q", crd.Group)
	}
	if crd.Version != "v1alpha1" {
		t.Errorf("expected Version=v1alpha1, got %q", crd.Version)
	}
	if crd.Plural != "websites" {
		t.Errorf("expected Plural=websites, got %q", crd.Plural)
	}
	if !crd.Namespaced {
		t.Error("expected Namespaced=true")
	}
}

func TestDiscoverCRD_BySingular(t *testing.T) {
	disc := fakeDiscovery(
		apiResource("demo.orkestra.io/v1alpha1", "Website", "websites", "website", true),
	)

	crd, err := inspect.DiscoverCRD(disc, "website")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if crd.Kind != "Website" {
		t.Errorf("expected Kind=Website, got %q", crd.Kind)
	}
}

func TestDiscoverCRD_ByKind_CaseInsensitive(t *testing.T) {
	disc := fakeDiscovery(
		apiResource("demo.orkestra.io/v1alpha1", "Website", "websites", "website", true),
	)

	tests := []string{"Website", "website", "WEBSITE", "wEbSiTe"}
	for _, name := range tests {
		crd, err := inspect.DiscoverCRD(disc, name)
		if err != nil {
			t.Errorf("DiscoverCRD(%q): unexpected error: %v", name, err)
			continue
		}
		if crd.Kind != "Website" {
			t.Errorf("DiscoverCRD(%q): expected Kind=Website, got %q", name, crd.Kind)
		}
	}
}

func TestDiscoverCRD_NotFound(t *testing.T) {
	disc := fakeDiscovery(
		apiResource("demo.orkestra.io/v1alpha1", "Website", "websites", "website", true),
	)

	_, err := inspect.DiscoverCRD(disc, "database")
	if err == nil {
		t.Fatal("expected error for unknown CRD, got nil")
	}
}

func TestDiscoverCRD_MultipleMatches(t *testing.T) {
	// Two groups that both have a "website" resource — ambiguous
	disc := fakeDiscovery(
		apiResource("demo.orkestra.io/v1alpha1", "Website", "websites", "website", true),
		apiResource("other.io/v1beta1", "Website", "websites", "website", true),
	)

	_, err := inspect.DiscoverCRD(disc, "website")
	if err == nil {
		t.Fatal("expected error for ambiguous CRD, got nil")
	}
}

func TestDiscoverCRD_SkipsSubresources(t *testing.T) {
	disc := fakeDiscovery(&metav1.APIResourceList{
		GroupVersion: "apps/v1",
		APIResources: []metav1.APIResource{
			{Name: "deployments", Kind: "Deployment", Namespaced: true},
			{Name: "deployments/scale", Kind: "Scale", Namespaced: true},       // subresource
			{Name: "deployments/status", Kind: "Deployment", Namespaced: true}, // subresource
		},
	})

	// Should find deployments but not deployments/scale
	crd, err := inspect.DiscoverCRD(disc, "deployments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if crd.Plural != "deployments" {
		t.Errorf("expected Plural=deployments, got %q", crd.Plural)
	}
}

func TestDiscoverCRD_GVR(t *testing.T) {
	disc := fakeDiscovery(
		apiResource("platform.myorg.io/v1alpha1", "PlatformNamespace", "platformnamespaces", "platformnamespace", false),
	)

	crd, err := inspect.DiscoverCRD(disc, "platformnamespace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if crd.GVR.Group != "platform.myorg.io" {
		t.Errorf("GVR.Group: expected %q, got %q", "platform.myorg.io", crd.GVR.Group)
	}
	if crd.GVR.Version != "v1alpha1" {
		t.Errorf("GVR.Version: expected %q, got %q", "v1alpha1", crd.GVR.Version)
	}
	if crd.GVR.Resource != "platformnamespaces" {
		t.Errorf("GVR.Resource: expected %q, got %q", "platformnamespaces", crd.GVR.Resource)
	}
	if crd.Namespaced {
		t.Error("expected Namespaced=false for cluster-scoped CRD")
	}
}

func TestDiscoverCRD_EmptyName(t *testing.T) {
	disc := fakeDiscovery()

	_, err := inspect.DiscoverCRD(disc, "")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}
