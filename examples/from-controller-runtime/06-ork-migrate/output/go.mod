module github.com/myorg/web-app-reconciler

go 1.22

require (
	github.com/orkspace/orkestra v0.7.6-25-gd71a9c07
	k8s.io/api v0.29.3
	k8s.io/apimachinery v0.29.3
	k8s.io/client-go v0.29.3
)

// Run: go mod tidy
// to resolve all indirect dependencies.
