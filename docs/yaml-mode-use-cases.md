# 🎯 **YAML Katalog: Use Cases & Scenarios**

The YAML mode in this framework transforms how operators are deployed, configured, and shared. By loading CRD configurations from local files or remote URLs, you unlock powerful operational patterns that are simply not possible with traditional Go-only approaches.

---

## 📦 **1. Centralized Operator Marketplace**

```bash
# Deploy a production-ready Prometheus operator instantly
export KATALOG_PATH=https://hub.orkestra.io/crds/prometheus-operator.yaml
./my-kontroller
```

**The Vision:** A community-driven hub where pre-configured CRD sets for popular tools (Prometheus, Cert-Manager, Istio, etc.) can be shared and consumed with zero setup.

**What it enables:**
- ✅ Discoverability – Browse available operator configurations
- ✅ Instant deployment – No need to write CRD definitions
- ✅ Version management – Pin to specific versions or track latest
- ✅ Quality assurance – Community-vetted configurations

**Example Hub Structure:**
```
https://hub.orkestra.io/crds/
├── monitoring/
│   ├── prometheus-operator.yaml
│   ├── grafana-operator.yaml
│   └── alertmanager.yaml
├── networking/
│   ├── istio.yaml
│   ├── nginx-ingress.yaml
│   └── cert-manager.yaml
└── storage/
    ├── rook-ceph.yaml
    └── minio-operator.yaml
```

---

## 🏢 **2. Organization-Wide Standardization**

```yaml
# https://internal-git.company.com/platform/crds/standard.yaml
crds:
  - name: project
    workers: 5
    namespace: platform
    # ... standard config across all teams
    
  - name: database
    workers: 3
    dependsOn: [project]
    # ... approved database configuration
```

**The Scenario:** Platform teams define **golden paths** – standardized CRD configurations that all development teams must use.

**Benefits:**
- ✅ Single source of truth – One YAML file governs all clusters
- ✅ Instant compliance – Changes propagate immediately
- ✅ Reduced drift – Every team runs identical configurations
- ✅ Audit readiness – Centralized view of all CRD definitions

**Deployment by teams:**
```bash
# Team A's cluster
export KATALOG_PATH=https://internal-git.company.com/platform/crds/standard.yaml
./kontroller

# Team B's cluster – same config, same behavior
export KATALOG_PATH=https://internal-git.company.com/platform/crds/standard.yaml
./kontroller
```

---

## 🌍 **3. Multi-Cluster Fleet Management**

```yaml
# Environment-specific configurations
https://git.company.com/fleet/
├── production/
│   └── crds.yaml     # 10 workers, aggressive retries
├── staging/
│   └── crds.yaml     # 5 workers, moderate settings
└── development/
    └── crds.yaml     # 2 workers, debug enabled
```

**The Scenario:** Organizations running hundreds of Kubernetes clusters need environment-specific tuning without maintaining separate kontroller builds.

**How it works:**
```bash
# Production fleet (high throughput)
export KATALOG_PATH=https://git.company.com/fleet/production/crds.yaml
./kontroller

# Staging fleet (moderate load)
export KATALOG_PATH=https://git.company.com/fleet/staging/crds.yaml
./kontroller

# Development fleet (minimal resources)
export KATALOG_PATH=https://git.company.com/fleet/dev/crds.yaml
./kontroller
```

**Production YAML example:**
```yaml
crds:
  - name: project
    workers: 20           # More workers for production load
    namespace: platform
    
  - name: application
    workers: 30
    dependsOn: [project]
    # Production-specific settings
```

---

## 🔄 **4. GitOps / Continuous Delivery**

```yaml
# https://github.com/myorg/infra/blob/main/crds/kontroller.yaml
# (raw GitHub URL)
```

**The Workflow:**

1. **Developer** updates CRD config in Git
2. **CI Pipeline** validates YAML syntax and dependencies
3. **Pull Request** reviewed by peers
4. **Merge to main** triggers automated deployment
5. **Kontrollers restart** (via rollout) and pick up new config
6. **Zero manual intervention**

**Benefits:**
- ✅ Version control – Full history of configuration changes
- ✅ Review process – Peer review before deployment
- ✅ Rollback capability – Revert to previous versions instantly
- ✅ Audit trail – Every change is tracked

**Example GitOps structure:**
```
infrastructure/
├── clusters/
│   ├── prod/
│   │   └── crds.yaml
│   ├── staging/
│   │   └── crds.yaml
│   └── dev/
│       └── crds.yaml
└── platform/
    └── crds/
        ├── database.yaml
        ├── messaging.yaml
        └── monitoring.yaml
```

---

## 🧪 **5. Canary Deployments & A/B Testing**

```yaml
# Control group (95% of clusters)
https://config.company.com/stable/crds.yaml

# Canary group (5% of clusters)
https://config.company.com/canary/crds.yaml
```

**The Scenario:** Safely test new CRD configurations on a small subset of clusters before rolling out everywhere.

**Canary YAML:**
```yaml
crds:
  - name: project
    workers: 25           # Testing higher concurrency
    # New experimental features
    
  - name: application
    workers: 35
    # Beta settings
```

**Deployment strategy:**
```bash
# 95% of clusters
export KATALOG_PATH=https://config.company.com/stable/crds.yaml

# 5% of clusters (canary)
export KATALOG_PATH=https://config.company.com/canary/crds.yaml
```

**Monitor metrics:**
- Compare reconciliation latency
- Track error rates
- Measure resource usage
- Gradual rollout based on confidence

---

## 🔧 **6. Dynamic Worker Scaling Based on Environment**

```yaml
# Small cluster (development)
crds:
  - name: project
    workers: 2
    
# Large cluster (production)
crds:
  - name: project
    workers: 20
```

**The Problem:** Different environments have different capacity needs. With YAML mode, you right-size resources per environment without rebuilding.

**Environment matrix:**

| Environment | Workers | Use Case |
|-------------|---------|----------|
| Development | 2 | Minimal resources, cost saving |
| Staging | 5 | Realistic load, pre-production |
| Production | 20 | High throughput, low latency |
| Peak season | 30 | Temporary scale-up |

---

## 👥 **7. Multi-Tenant / Team Isolation**

```yaml
# https://config.company.com/teams/frontend/crds.yaml
crds:
  - name: frontend-app
    workers: 5
    namespace: frontend-team
    
# https://config.company.com/teams/backend/crds.yaml
crds:
  - name: backend-api
    workers: 10
    namespace: backend-team
    dependsOn: [database]
```

**The Scenario:** Each team manages their own CRD configurations, stored in their own repositories, without interfering with other teams.

**Team autonomy:**
- ✅ Each team controls their own CRD definitions
- ✅ No central bottleneck
- ✅ Team-specific worker allocations
- ✅ Independent versioning

---

## 🔐 **8. Compliance & Audit Trails**

```yaml
# Corporate proxy with authentication
export KATALOG_PATH=https://corporate-git.company.com/approved/crds.yaml
```

**Security benefits:**
- ✅ **Access control** – Remote files behind corporate auth
- ✅ **Audit logs** – Every access is recorded
- ✅ **Approved sources** – Only allow registries from trusted domains
- ✅ **Immutable history** – Git provides complete audit trail
- ✅ **Compliance** – Configurations follow approved patterns

**Corporate requirements met:**
- SOX compliance
- HIPAA requirements
- PCI DSS standards
- Internal security policies

---

## 🌐 **9. Edge Deployments & IoT**

```yaml
# Lightweight config for resource-constrained edge devices
crds:
  - name: sensor
    workers: 1
    namespace: edge
    
  - name: actuator
    workers: 1
    dependsOn: [sensor]
```

**The Scenario:** Thousands of edge devices with limited resources need optimized configurations. YAML mode allows central management of distributed fleets.

**Edge benefits:**
- ✅ Small footprint – Minimal YAML instead of full Go code
- ✅ Remote updates – Push new configs without SSH
- ✅ Gradual rollout – Update edge devices in phases
- ✅ Offline capability – Cache katalog locally

---

## 🎯 **10. Partner & Customer Integrations**

```yaml
# Expose to external partners
https://api.company.com/partners/acme/crds.yaml
```

**The Scenario:** Provide customized operator configurations to partners without sharing internal code or giving direct cluster access.

**Partner workflow:**
1. Partner receives authenticated URL
2. Kontroller fetches partner-specific YAML
3. Partner runs their own instance with your configuration
4. You control the CRD definitions remotely

---

## 📊 **Comparison: Local vs Remote YAML**

| Feature | Local YAML | Remote YAML |
|---------|------------|-------------|
| **Configuration source** | File system | Any HTTPS URL |
| **Update mechanism** | File replace | GitOps / CI/CD |
| **Multi-cluster sync** | Manual copy | Centralized |
| **Version control** | Optional | Native (Git) |
| **Access control** | Filesystem permissions | Corporate auth |
| **Audit trail** | Limited | Full Git history |
| **Team autonomy** | Per cluster | Per repository |
| **Canary testing** | Manual | Automatic with URLs |

---

## 🏁 **The Ultimate Vision**

With remote YAML support, your kontroller becomes:

> *"A single binary that can be configured to manage ANY set of CRDs, with ANY dependencies, on ANY cluster, controlled by ANY Git repository, deployed by ANY team, audited by ANY compliance officer."*


# 🧩 **11. Why YAML Mode Exists (The Philosophy)**

Go mode is perfect for **framework developers**.  
YAML mode is perfect for **platform operators**.

The two modes serve different audiences:

| Mode | Audience | Strength |
|------|----------|----------|
| **Go Mode** | Framework authors | Full control, compile‑time safety |
| **YAML Mode** | Platform teams, SREs, DevOps, GitOps | Dynamic configuration, no rebuilds |

**YAML mode exists because configuration is not code.**  
It changes faster, is owned by different teams, and must be deployable without recompiling binaries.

This separation of concerns is what makes your framework scalable across organizations.

---

# 🔐 **12. Security Model & Governance**

YAML mode enables a secure, governed configuration pipeline.

### **Domain Allowlisting**
Control which sources are trusted:
```bash
export KATALOG_PATH_ALLOWED_DOMAINS="*.company.com,*.github.com"
```

### **Checksum Verification**
Prevent tampering with SHA256 verification:
```yaml
# crds.yaml.sha256
8f3c9a1b2e...  crds.yaml
```

### **Immutable Versioning**
Pin to exact Git commit SHAs for reproducibility:
```bash
export KATALOG_PATH=https://raw.githubusercontent.com/org/repo/8f3c9a1/crds.yaml
```

### **Audit Logging**
Every remote fetch is logged:
```json
{
  "level":"info",
  "time":1773117835,
  "url":"https://hub.orkestra.io/crds/prometheus.yaml",
  "checksum":"8f3c9a1b2e...",
  "message":"katalog loaded"
}
```

**Governance Benefits:**
- ✅ Centralized control over approved CRDs
- ✅ Compliance with internal security policies
- ✅ Full traceability of configuration changes
- ✅ Instant revocation of compromised configurations

---

# 🧪 **13. Versioning Strategy & Release Channels**

YAML mode supports multiple release channels, similar to Kubernetes itself:

### **Stable (Production Ready)**
```
https://hub.orkestra.io/crds/prometheus/stable.yaml
```

### **Beta (Pre-release Testing)**
```
https://hub.orkestra.io/crds/prometheus/beta.yaml
```

### **Nightly (Latest Development)**
```
https://hub.orkestra.io/crds/prometheus/nightly.yaml
```

### **Pinned Version (Deterministic)**
```
https://hub.orkestra.io/crds/prometheus/v1.4.2.yaml
```

### **Git Commit SHA (Maximum Reproducibility)**
```
https://raw.githubusercontent.com/org/repo/8f3c9a1/crds.yaml
```

**This enables:**
- ✅ Deterministic deployments – Same SHA = same behavior
- ✅ Reproducible environments – Dev/staging/prod match exactly
- ✅ Safe rollbacks – Revert to previous SHA instantly
- ✅ Controlled experimentation – Test beta without affecting production

---

# 🧑‍💻 **14. Local Development Workflow**

YAML mode isn't just for production — it's perfect for developers too.

### **Local Iteration Loop**
```bash
# Point to local YAML file
export KATALOG_PATH=./crds/dev.yaml
go run ./cmd

# Edit dev.yaml, restart kontroller, see changes instantly
```

### **Team Collaboration**
- Developers share YAML configs instead of Go code
- Frontend engineers can define CRDs without learning Go
- Platform teams distribute standard configurations via Git

### **Fast Onboarding**
New team members can contribute by editing YAML, not navigating complex Go codebases.

**This makes the framework accessible to non-Go engineers** – SREs, DevOps, and platform engineers can all participate.

---

# 🧱 **15. Architecture: YAML Mode as a Control Plane**

YAML mode effectively turns your kontroller into a **mini control plane**:

| Komponent | Analogy |
|----------|---------|
| CRDs | Packages |
| Dependencies | Service graphs |
| Workers | Execution units |
| YAML | Desired state |

This mirrors proven architectures:

- **Crossplane** – Packages as CRDs
- **Kubernetes API aggregation** – Multiple APIs served by one apiserver
- **Helm charts** – Parameterized package definitions
- **OperatorHub.io** – Curated operator catalog

**But with a critical differentiator:**  
Runtime dependency orkestration that none of those provide.

---

# 🧲 **16. Offline & Air‑Gapped Environments**

Many enterprises run clusters in restricted environments:

- Air‑gapped networks
- Secure enclaves
- Private datacenters
- Military or regulated environments

YAML mode supports them all:

### **Local File Mode**
```bash
export KATALOG_PATH=/etc/kontroller/cache/crds.yaml
```

### **Mirrored Katalog**
```bash
export KATALOG_PATH=https://mirror.company.com/crds.yaml
```

### **Bundle Mode**
Package all CRDs into a single tarball:
```bash
kontroller --bundle crds.tar.gz
```

### **Pre‑loaded Cache**
Pre-download all configurations during build:
```dockerfile
ADD https://hub.orkestra.io/crds/production.yaml /etc/kontroller/crds.yaml
```

**This makes the framework viable in the most restrictive environments** – government, finance, healthcare, and defense.

---

# 🧭 **17. Self‑Validation & Safety Guarantees**

Before starting, orkestra validates:

| Validation | What It Checks |
|-----------|----------------|
| **CRD existence** | Are all required CRDs installed in the cluster? |
| **API compatibility** | Does the API server support required versions? |
| **Dependency cycles** | Are there circular dependencies (A → B → A)? |
| **Worker configuration** | Are worker counts positive integers? |
| **Namespace scoping** | Do namespaced CRDs have valid namespaces? |
| **GVK correctness** | Do Group/Version/Kinds match registered schemes? |
| **Remote availability** | Is the remote katalog URL accessible? |

If anything is invalid, orkestra **fails fast** with a clear error:

```bash
Error: circular dependency detected: project → application → project
```

**This prevents:**
- ❌ Partial startup with missing CRDs
- ❌ Silent misconfiguration
- ❌ Undefined behavior at runtime
- ❌ Hard-to-debug production issues

---

# 🧨 **18. Progressive Delivery for CRD Configurations**

YAML mode enables sophisticated rollout strategies:

### **Blue/Green Deployment**
```bash
# Blue (current)
cluster-1: export KATALOG_PATH=https://config.company.com/v1.yaml

# Green (candidate)  
cluster-2: export KATALOG_PATH=https://config.company.com/v2.yaml
```

### **Weighted Rollout**
```bash
# 90% of clusters on v1, 10% on v2
clusters A–M → v1
clusters N–Z → v2
```

### **Automated Rollback**
If error rate spikes after deployment:
1. Detect anomaly via metrics
2. Revert to previous YAML URL
3. Restart kontroller
4. Restore stable behavior

**This is the same pattern used by:**
- Service meshes (Istio, Linkerd)
- Deployment kontrollers (Argo Rollouts)
- Feature flag systems (LaunchDarkly)

---

# 🧠 **19. Observability & Metrics Integration**

YAML mode integrates seamlessly with Prometheus metrics:

```
# HELP katalog_fetch_total Total number of katalog fetches
# HELP katalog_PATH_fetch_duration Duration of katalog fetches
# HELP crd_config_valid Configuration validation status (1=valid, 0=invalid)
```

This allows platform teams to:

- 📊 **Compare configurations** – Which version performs better?
- ⚙️ **Tune worker counts** – Right-size based on load
- 🔍 **Detect regressions** – Error rate spikes after YAML change
- ✅ **Validate canaries** – Compare metrics across channels
- 📈 **Trend analysis** – How does configuration affect performance?

**It turns YAML mode into a measurable, observable system.**

---

# 🛡 **20. Disaster Recovery & Cluster Migration**

YAML mode makes cluster migration trivial:

```bash
# Rebuild a cluster from scratch
export KATALOG_PATH=https://git.company.com/platform/crds/prod.yaml
./kontroller --kubekonfig new-cluster.yaml
```

**Use cases:**

| Scenario | Command |
|----------|---------|
| **Cluster rebuild** | Point to same YAML, restart |
| **DR failover** | Same YAML, different cluster |
| **Region migration** | Same YAML, different region |
| **Cluster replacement** | Same YAML, new cluster |

**This enables:**
- ✅ Infrastructure as code – CRD configs in Git
- ✅ Disaster recovery – Rebuild entire platform from YAML
- ✅ Cross-region replication – Consistent config everywhere
- ✅ Zero-downtime migration – Move workloads without reconfiguring

---

## 🏁 **Summary: Why YAML Mode Matters**

| Capability | What It Enables |
|-----------|-----------------|
| **Configuration as data** | No rebuilds for CRD changes |
| **Remote loading** | Centralized management, GitOps |
| **Dependency graphs** | Safe startup/shutdown ordering |
| **Per-CRD workers** | Fine-grained resource control |
| **Security & governance** | Domain allowlisting, checksums |
| **Versioning** | Stable/beta/nightly channels |
| **Air-gapped support** | Works anywhere |
| **Self-validation** | Fail fast, fail safe |
| **Progressive delivery** | Canary, blue/green, rollback |
| **Observability** | Metrics for every operation |
| **Disaster recovery** | Rebuild from YAML alone |

**YAML mode transforms your kontroller from a tool into a platform.** 🚀