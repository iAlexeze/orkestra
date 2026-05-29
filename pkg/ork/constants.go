package ork

const (
	// Orkestra is the Helm release name and the prefix for all Orkestra resources.
	Orkestra = "orkestra"

	// OrkestraRuntime is the name of the runtime Deployment.
	OrkestraRuntime = "orkestra-runtime"

	// OrkestraNamespace is the Kubernetes namespace Orkestra deploys into.
	OrkestraNamespace = "orkestra-system"

	// OrkestraChartRepo is the Helm repository URL for the Orkestra chart.
	OrkestraChartRepo = "https://orkspace.github.io/orkestra"

	// OrkestraChartName is the chart name within the repository.
	OrkestraChartName = "orkestra"

	// OrkestraControlCenter is the name of the Control Center Deployment.
	OrkestraControlCenter = "orkestra-cc"

	// OrkestraControlCenterPort is the default Control Center HTTP port.
	OrkestraControlCenterPort = "8081"
)
