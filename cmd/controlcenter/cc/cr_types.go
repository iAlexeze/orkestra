package controlcenter

// ─────────────────────────────────────────────────────────────────────────────
// CR types — mirror the JSON shapes returned by Orkestra's
// /katalog/{crd}/cr and /katalog/{crd}/cr/{namespace}/{name} endpoints.
// ─────────────────────────────────────────────────────────────────────────────

// CRSummary is one row in the CR list — one line of kubectl get.
type CRSummary struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Phase       string `json:"phase,omitempty"`
	Ready       bool   `json:"ready"`
	ReadyReason string `json:"readyReason,omitempty"`
	Age         string `json:"age"`
	Generation  int64  `json:"generation"`
}

// CRListResponse is returned by GET /katalog/{crd}/cr.
type CRListResponse struct {
	CRD   string      `json:"crd"`
	GVK   string      `json:"gvk"`
	Total int         `json:"total"`
	Items []CRSummary `json:"items"`
}

// ChildSummary is a condensed view of one child resource.
type ChildSummary struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace,omitempty"`
	Kind      string                 `json:"kind"`
	Status    map[string]interface{} `json:"status,omitempty"`
	Ready     bool                   `json:"ready"`
}

// CRDetailResponse is returned by GET /katalog/{crd}/cr/{namespace}/{name}.
type CRDetailResponse struct {
	Name              string                  `json:"name"`
	Namespace         string                  `json:"namespace,omitempty"`
	Generation        int64                   `json:"generation"`
	CreationTimestamp string                  `json:"creationTimestamp"`
	Labels            map[string]string       `json:"labels,omitempty"`
	Annotations       map[string]string       `json:"annotations,omitempty"`
	Ready             bool                    `json:"ready"`
	ReadyReason       string                  `json:"readyReason,omitempty"`
	ReadyMessage      string                  `json:"readyMessage,omitempty"`
	Status            map[string]interface{}  `json:"status,omitempty"`
	Children          map[string]ChildSummary `json:"children,omitempty"`
	EventsEndpoint    string                  `json:"eventsEndpoint"`
}

// CREvent is one Kubernetes event involving this CR.
type CREvent struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
	Source    string `json:"source"`
	Object    string `json:"object"`
	Count     int32  `json:"count"`
	FirstSeen string `json:"firstSeen"`
	LastSeen  string `json:"lastSeen"`
	Age       string `json:"age"`
}

// CREventsResponse is returned by GET /katalog/{crd}/cr/{namespace}/{name}/events.
type CREventsResponse struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace,omitempty"`
	Total     int       `json:"total"`
	Events    []CREvent `json:"events"`
}

// ─────────────────────────────────────────────────────────────────────────────
// View models — assembled from API responses for template rendering.
// ─────────────────────────────────────────────────────────────────────────────

// CRListView is the data passed to cr_list.html.
type CRListView struct {
	KatalogName string // e.g. "demo"
	Instance    string // Orkestra instance URL
	CRDName     string // e.g. "pipeline"
	GVK         string // e.g. "demo.orkestra.io/v1alpha1, Kind=Pipeline"
	Total       int
	Items       []CRSummary
	BackURL     string // back to /katalog/{crd}
}

// CRDetailView is the data passed to cr_detail.html.
type CRDetailView struct {
	KatalogName string // e.g. "demo"
	Instance    string
	CRDName     string
	CR          CRDetailResponse
	Events      []CREvent
	EventTotal  int
	Phase       string // extracted from status.phase for convenience
	BackURL     string // back to /katalog/{crd}/cr
}
