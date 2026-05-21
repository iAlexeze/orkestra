package konfig

import "time"

// Port returns the configured port for the health server.
func (h *healthServer) Port() string {
	return h.port
}

// SetPort sets the configured port for the health server.
func (h *healthServer) SetPort(v string) {
	h.port = v
}

// ReadTimeout returns the read timeout for the health server.
func (h *healthServer) ReadTimeout() time.Duration {
	return h.readTimeout
}

// SetReadTimeout sets the read timeout for the health server.
func (h *healthServer) SetReadTimeout(v time.Duration) {
	h.readTimeout = v
}

// WriteTimeout returns the write timeout for the health server.
func (h *healthServer) WriteTimeout() time.Duration {
	return h.writeTimeout
}

// SetWriteTimeout sets the write timeout for the health server.
func (h *healthServer) SetWriteTimeout(v time.Duration) {
	h.writeTimeout = v
}

// KubekonfigPath returns the path to the kubeconfig file for the cluster.
func (c *clusterKonfig) KubekonfigPath() string {
	return c.kubekonfigPath
}

// SetKubekonfigPath sets the path to the kubeconfig file for the cluster.
func (c *clusterKonfig) SetKubekonfigPath(v string) {
	c.kubekonfigPath = v
}

// MasterURL returns the API server master URL for the cluster.
func (c *clusterKonfig) MasterURL() string {
	return c.masterURL
}

// SetMasterURL sets the API server master URL for the cluster.
func (c *clusterKonfig) SetMasterURL(v string) {
	c.masterURL = v
}

// Name returns the logical name of the cluster.
func (c *clusterKonfig) Name() string {
	return c.name
}

// SetName sets the logical name of the cluster.
func (c *clusterKonfig) SetName(v string) {
	c.name = v
}

// Namespace returns the namespace used by the cluster components.
func (c *clusterKonfig) Namespace() string {
	return c.namespace
}

// SetNamespace sets the namespace used by the cluster components.
func (c *clusterKonfig) SetNamespace(v string) {
	c.namespace = v
}
