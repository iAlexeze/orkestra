package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Katalog struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"metadata"`
	Spec struct {
		CRDs map[string]CRDEntry `yaml:"crds"`
	} `yaml:"spec"`
}

type CRDEntry struct {
	APITypes struct {
		Group   string `yaml:"group"`
		Version string `yaml:"version"`
		Kind    string `yaml:"kind"`
		Plural  string `yaml:"plural"`
	} `yaml:"apiTypes"`
	OperatorBox struct {
		Reconciler struct {
			Workers int    `yaml:"workers"`
			Resync  string `yaml:"resync"`
		} `yaml:"reconciler"`
		OnCreate struct {
			Deployments []Deployment `yaml:"deployments"`
			Services    []Service    `yaml:"services"`
			Ingresses   []Ingress    `yaml:"ingresses,omitempty"`
		} `yaml:"onCreate"`
	} `yaml:"operatorBox"`
}

type Deployment struct {
	Name      string `yaml:"name"`
	Image     string `yaml:"image"`
	Replicas  string `yaml:"replicas"`
	Port      int    `yaml:"port"`
	Reconcile bool   `yaml:"reconcile"`
}

type Service struct {
	Name       string `yaml:"name"`
	Port       int    `yaml:"port"`
	TargetPort int    `yaml:"targetPort"`
	Reconcile  bool   `yaml:"reconcile"`
}

type Ingress struct {
	Name         string `yaml:"name"`
	Host         string `yaml:"host"`
	ServiceName  string `yaml:"serviceName"`
	ServicePort  int    `yaml:"servicePort"`
	ClassName string `yaml:"className"`
	Reconcile    bool   `yaml:"reconcile"`
}

func generateKatalog(appName, image string, port, replicas int, ingressHost string) *Katalog {
	k := &Katalog{}
	k.APIVersion = "orkestra.orkspace.io/v1"
	k.Kind = "Katalog"
	k.Metadata.Name = appName + "-operator"
	k.Metadata.Description = fmt.Sprintf("Generated Katalog for %s", appName)

	crd := CRDEntry{}
	crd.APITypes.Group = "web.orkestra.io"
	crd.APITypes.Version = "v1alpha1"
	crd.APITypes.Kind = "WebApp"
	crd.APITypes.Plural = "webapps"
	crd.OperatorBox.Reconciler.Workers = 2
	crd.OperatorBox.Reconciler.Resync = "30s"

	// Deployment
	crd.OperatorBox.OnCreate.Deployments = []Deployment{
		{
			Name:      "{{ .metadata.name }}-deploy",
			Image:     image,
			Replicas:  fmt.Sprintf("%d", replicas),
			Port:      port,
			Reconcile: true,
		},
	}

	// Service
	crd.OperatorBox.OnCreate.Services = []Service{
		{
			Name:       "{{ .metadata.name }}-svc",
			Port:       port,
			TargetPort: port,
			Reconcile:  true,
		},
	}

	// Ingress (optional)
	if ingressHost != "" {
		crd.OperatorBox.OnCreate.Ingresses = []Ingress{
			{
				Name:         "{{ .metadata.name }}-ingress",
				Host:         ingressHost,
				ServiceName:  "{{ .metadata.name }}-svc",
				ServicePort:  port,
				ClassName: "nginx",
				Reconcile:    true,
			},
		}
	}

	k.Spec.CRDs = map[string]CRDEntry{appName: crd}
	return k
}

func main() {
	appName := flag.String("name", "myapp", "Application name")
	image := flag.String("image", "nginx:1.25", "Container image")
	port := flag.Int("port", 8080, "Container port")
	replicas := flag.Int("replicas", 2, "Number of replicas")
	ingressHost := flag.String("ingress", "", "Ingress host (optional)")
	flag.Parse()

	k := generateKatalog(*appName, *image, *port, *replicas, *ingressHost)

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(k); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
