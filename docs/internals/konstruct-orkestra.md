# konstructOrkestra (Internal Wiring)

`konstructOrkestra` is the internal function that wires the entire Orkestra
runtime. It does not start anything — it only constructs the dependency graph
of components.

This separation allows:

- deterministic startup  
- deterministic shutdown  
- clean testing  
- clean dependency injection  
- predictable lifecycle  

---

## Responsibilities

### 1. Load + validate Katalog
- Merge sources via Komposer  
- Apply defaults  
- Validate schema  
- Build CRD entries  

### 2. Build scheme registry
Typed CRDs register their `AddToScheme` functions here.

### 3. Create core components
- HealthServer  
- Kubeclient  
- EventRecorder  
- QueueRegistry  
- Default Workqueue  

### 4. Register REST client constructors
Typed CRDs only.

### 5. Build SharedInformerFactory
One informer per CRD.

### 6. Register CRDs in KontrollerRegistry
Maps GVK → ReconcilerFactory.

### 7. Build dependency graph
Used by DependencyKontroller.

### 8. Create DependencyKontroller
But do not start it yet.

### 9. Register health + Katalog endpoints
Before the server starts.

---

## Output

`konstructOrkestra` returns a struct containing:

- kubeclient  
- event recorder  
- informer factory  
- kontroller registry  
- dependency kontroller  
- health server  
- queue registry  
- runtime katalog  

This struct is passed to `Orkestra.Start()`.
