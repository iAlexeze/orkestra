package controlcenter

// KatalogListData is the data for the main index page
type KatalogListData struct {
	Katalogs        []KatalogSummary
	TotalKatalogs   int
	HealthyKatalogs int
	TotalCRDs       int
	TotalWorkers    int
	TotalResources  int
	AnyHealthy      bool
	OrkestraURLs    string
}

// KatalogSummary is a summary of a Katalog for the list view
type KatalogSummary struct {
	Name           string
	Description    string
	Version        string
	Healthy        bool
	TotalCRDs      int
	HealthyCRDs    int
	TotalWorkers   int
	TotalResources int
}

// KatalogData is the data for the Katalog dashboard view
type KatalogData struct {
	CRDs               []CRDSummary
	TotalCRDs          int
	OrkReady           bool
	DeletionProtection bool
	TotalWorkers       int
	TotalResources     int
	HealthyCount       int
	KatalogName        string
	KatalogDescription string
	KatalogHealthy     bool
	KatalogVersion     string
	KatalogAuthor      string
	KatalogLicense     string
	DegradedReason     string
	StatusCounts       StatusCounts
}

// IndexData is the data for the main page
type IndexData struct {
	Katalogs        []KatalogSummary
	TotalKatalogs   int
	HealthyKatalogs int
	TotalCRDs       int
	TotalWorkers    int
	TotalResources  int
	AnyHealthy      bool
	OrkestraURLs    string
}

// StatusCounts tracks CRD health counts
type StatusCounts struct {
	Healthy  int `json:"healthy"`
	Degraded int `json:"degraded"`
	Started  int `json:"started"`
	Pending  int `json:"pending"`
}

// RBACInfo is the data for the main page
type RBACInfo struct {
	Rules      []RBACRule `json:"rules"`
	Summary    string     `json:"summary"`
	TotalRules int        `json:"totalRules"`
}

type RBACRule struct {
	APIGroups []string `json:"apiGroups,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Verbs     []string `json:"verbs,omitempty"`
	// Human-readable description
	Description string `json:"description,omitempty"`
}

// KatalogResponse is the response from the /katalog endpoint
type KatalogResponse struct {
	Total              int          `json:"total"`
	TotalEnabled       int          `json:"totalEnabled"`
	Healthy            bool         `json:"healthy"`
	Status             int          `json:"status"`
	OrkReady           bool         `json:"OrkReady"`
	DeletionProtection bool         `json:"deletionProtection"`
	CRDs               []CRDSummary `json:"crds"`
	Name               string       `json:"name,omitempty"`
	Version            string       `json:"version,omitempty"`
	Author             string       `json:"author,omitempty"`
	Description        string       `json:"description,omitempty"`
	DegradedReason     string       `json:"degradedReason,omitempty"`
	StatusCounts       StatusCounts `json:"statusCounts"`
	License            string       `json:"license,omitempty"`
}

// CRDSummary is a summary of a CRD
type CRDSummary struct {
	Name                     string   `json:"name"`
	State                    string   `json:"state"` // "healthy", "started", "pending", "degraded"
	Healthy                  bool     `json:"healthy"`
	Started                  bool     `json:"started"`
	Pending                  bool     `json:"pending"`
	Workers                  int      `json:"workers"`
	WorkersActive            int      `json:"workersActive"`
	DependsOn                []string `json:"dependsOn"`
	WorkersSource            string   `json:"workersSource"`
	QueueDepth               int      `json:"queueDepth"`
	MaxQueueDepth            int      `json:"maxQueueDepth"`
	ResourceCount            int      `json:"resourceCount"`
	ErrorRate                float64  `json:"errorRate"`
	Uptime                   string   `json:"uptime"`
	RBACCount                int      `json:"rbacCount,omitempty"`
	HasUnhealthyDependencies bool     `json:"hasUnhealthyDependencies"`
	DeletionProtection       bool     `json:"deletionProtection"`
}

// CRDHealth is the response from the /katalog/{crd}/health endpoint
type CRDHealth struct {
	Name                     string                      `json:"name"`
	State                    string                      `json:"state"`
	Healthy                  bool                        `json:"healthy"`
	Started                  bool                        `json:"started"`
	Pending                  bool                        `json:"pending"`
	StartedAt                string                      `json:"startedAt"`
	Uptime                   string                      `json:"uptime"`
	QueueDepth               int                         `json:"queueDepth"`
	ErrorRate                float64                     `json:"errorRate"`
	ConsecutiveFails         int                         `json:"consecutiveFails"`
	TotalReconciles          int                         `json:"totalReconciles"`
	ResourceCount            int                         `json:"resourceCount"`
	LastError                string                      `json:"lastError"`
	LastReconcile            string                      `json:"lastReconcile"`
	HasUnhealthyDependencies bool                        `json:"hasUnhealthyDependencies"`
	Dependencies             map[string]DependencyStatus `json:"dependencies,omitempty"`
}

type DependencyStatus struct {
	Name                string `json:"name"`
	State               string `json:"state"`
	Condition           string `json:"condition"`
	Satisfied           bool   `json:"satisfied"`
	LastCheck           string `json:"lastCheck,omitempty"`
	AcceptableCondition string `json:"acceptableCondition"` // "started", "healthy", "ready"
}

// CRDInfo is the response from the /katalog/{crd} endpoint
type CRDInfo struct {
	Name                     string                 `json:"name"`
	Description              string                 `json:"description"`
	Mode                     string                 `json:"mode"`
	GVK                      string                 `json:"gvk"`
	GVR                      string                 `json:"gvr"`
	Namespaced               bool                   `json:"namespaced"`
	Namespace                string                 `json:"namespace"`
	DependsOn                []string               `json:"dependsOn"`
	Workers                  int                    `json:"workers"`
	WorkersActive            int32                  `json:"workersActive"`
	WorkersIdle              int32                  `json:"workersIdle"`
	WorkersProcessing        int32                  `json:"workersProcessing"`
	WorkerDetails            map[string]string      `json:"workerDetails,omitempty"`
	WorkersSource            string                 `json:"workersSource"`
	Resync                   string                 `json:"resync"`
	ResyncSource             string                 `json:"resyncSource"`
	QueueDepth               int                    `json:"queueDepth"`
	MaxQueueDepth            int                    `json:"maxQueueDepth"`
	MaxQueueDepthSource      string                 `json:"maxQueueDepthSource"`
	ResourceCount            int                    `json:"resourceCount"`
	TotalReconciles          int                    `json:"totalReconciles"`
	Reconciler               map[string]interface{} `json:"reconciler"`
	Healthy                  bool                   `json:"healthy"`
	Started                  bool                   `json:"started"`
	ErrorRate                float64                `json:"errorRate"`
	Conversion               *ConversionStats       `json:"conversion"`
	Admission                *AdmissionStats        `json:"admission"`
	Protection               *ProtectionStats       `json:"protection,omitempty"`
	RBAC                     RBACInfo               `json:"rbac,omitempty"`
	HasUnhealthyDependencies bool                   `json:"hasUnhealthyDependencies"`
}

// ConversionStats contains version conversion metrics
type ConversionStats struct {
	Enabled      bool    `json:"enabled"`
	Total        int     `json:"total"`
	Success      int     `json:"success"`
	Failures     int     `json:"failures"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	P95LatencyMs float64 `json:"p95LatencyMs"`
}

// ProtectionStats contains deletion protection status
type ProtectionStats struct {
	Enabled bool `json:"enabled"`
	Total   int  `json:"total"`
	Blocked int  `json:"blocked"`
	Allowed int  `json:"allowed"`
}

// AdmissionStats contains admission webhook metrics
type AdmissionStats struct {
	WebhooksEnabled   bool    `json:"webhooksEnabled"`
	ValidationTotal   int     `json:"validationTotal"`
	ValidationAllowed int     `json:"validationAllowed"`
	ValidationDenied  int     `json:"validationDenied"`
	ValidationWarned  int     `json:"validationWarned"`
	ValAvgLatencyMs   float64 `json:"valAvgLatencyMs"`
	ValP95LatencyMs   float64 `json:"valP95LatencyMs"`
	ValMaxLatencyMs   float64 `json:"valMaxLatencyMs"`
	MutationTotal     int     `json:"mutationTotal"`
	MutationApplied   int     `json:"mutationApplied"`
	MutationSkipped   int     `json:"mutationSkipped"`
	MutAvgLatencyMs   float64 `json:"mutAvgLatencyMs"`
	MutP95LatencyMs   float64 `json:"mutP95LatencyMs"`
	MutMaxLatencyMs   float64 `json:"mutMaxLatencyMs"`
}

// CRDDetail combines health and info for a single CRD
type CRDDetail struct {
	State                    string                      `json:"state"`
	StartedAt                string                      `json:"startedAt"`
	Uptime                   string                      `json:"uptime"`
	ConsecutiveFails         int                         `json:"consecutiveFails"`
	LastError                string                      `json:"lastError"`
	LastReconcile            string                      `json:"lastReconcile"`
	StartedAgo               string                      `json:"startedAgo"`
	LastReconcileAgo         string                      `json:"lastReconcileAgo"`
	Name                     string                      `json:"name"`
	Description              string                      `json:"description"`
	Mode                     string                      `json:"mode"`
	GVK                      string                      `json:"gvk"`
	GVR                      string                      `json:"gvr"`
	Namespaced               bool                        `json:"namespaced"`
	Namespace                string                      `json:"namespace"`
	DependsOn                []string                    `json:"dependsOn"`
	HasUnhealthyDependencies bool                        `json:"hasUnhealthyDependencies"`
	Dependencies             map[string]DependencyStatus `json:"dependencies,omitempty"`
	Workers                  int                         `json:"workers"`
	WorkersActive            int                         `json:"workersActive"`
	WorkersIdle              int                         `json:"workersIdle"`
	WorkersProcessing        int                         `json:"workersProcessing"`
	WorkerDetails            map[string]string           `json:"workerDetails,omitempty"`
	WorkersSource            string                      `json:"workersSource"`
	Resync                   string                      `json:"resync"`
	ResyncSource             string                      `json:"resyncSource"`
	QueueDepth               int                         `json:"queueDepth"`
	MaxQueueDepth            int                         `json:"maxQueueDepth"`
	MaxQueueDepthSource      string                      `json:"maxQueueDepthSource"`
	ResourceCount            int                         `json:"resourceCount"`
	TotalReconciles          int                         `json:"totalReconciles"`
	Reconciler               map[string]interface{}      `json:"reconciler"`
	Healthy                  bool                        `json:"healthy"`
	Started                  bool                        `json:"started"`
	Pending                  bool                        `json:"pending"`
	ErrorRate                float64                     `json:"errorRate"`
	Conversion               *ConversionStats            `json:"conversion"`
	Admission                *AdmissionStats             `json:"admission"`
	Protection               *ProtectionStats            `json:"protection,omitempty"`
	RBAC                     RBACInfo                    `json:"rbac,omitempty"`
	RBACCount                int                         `json:"rbacCount,omitempty"`
}

// TODO: Future
type SimpleSystemMetrics struct {
	Goroutines int     `json:"goroutines"`
	Threads    int     `json:"threads"`
	Gomaxprocs int     `json:"gomaxprocs"`
	HeapAlloc  uint64  `json:"heapAlloc"`
	HeapSys    uint64  `json:"heapSys"`
	GCPauseAvg float64 `json:"gcPauseAvg"`
}
