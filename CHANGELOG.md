# **CHANGELOG — ONCOP Integration (Orkestra Native Cross‑Operator Protocol)**

### **Added — ONCOP v1 (Orkestra Native Cross‑Operator Protocol)**  
Introduced ONCOP as the unified, typed, cross‑operator observation protocol for Orkestra. ONCOP replaces ad‑hoc HTTP integrations and hard‑coded URLs with a declarative, URL‑inferable, cache‑aware protocol used across autoscaling, status fields, and template resolution.

Key components:

- **Typed observation surfaces**  
  Added first‑class ONCOP types:  
  `metrics`, `health`, `cr`, `info`, `events`  
  Each type maps to a deterministic URL shape under `/katalog/<crd>`.

- **URL inference engine**  
  Implemented `BuildONCOPURL` to construct ONCOP URLs from `CrossCRDDeclaration` using:  
  `source.host`, `source.type`, `crd`, `selector.namespace`, `selector.name`.

- **Cross‑operator resolver integration**  
  Updated `readCross()` to support ONCOP host‑based reads as Path 2, after informer registry and before raw endpoint fallback.  
  Responses injected into `.cross.<as>` for templates, autoscale conditions, and status fields.

- **New ONCOP type: `cr`**  
  Added `type: cr` for CR‑specific detail (`status`, `spec`, `children`, `metrics`).  
  Distinguishes CR detail from CRD‑level `info`.

- **Autoscaler ONCOP support**  
  Autoscale conditions now resolve `cross.<crd>.metrics.*` via ONCOP metrics endpoint with optional caching (`cacheFor:`).

- **Resolver enhancements**  
  Added `ParseCrossField` and extraction helpers (`ExtractCrossCRD`, `ExtractCrossCategory`, `ExtractCrossFieldName`, `ExtractCrossNamespace`) to unify cross‑field parsing.

- **Fallback semantics**  
  Resolution priority formalised as:  
  `informer registry → ONCOP host → raw endpoint → empty result`.

- **Cross‑binary caching**  
  Added per‑source caching for ONCOP responses to avoid repeated remote calls.

### **Impact**  
ONCOP enables consistent, declarative, cross‑operator observation across Orkestra.  
Autoscalers, status fields, and templates now consume cross‑operator data without bespoke integrations or hard‑coded URLs.  
Operators implementing ONCOP become first‑class participants in the Orkestra ecosystem.
