# Dynamic vs Typed CRDs (Internals)

Orkestra supports two CRD modes:

- **Dynamic CRDs** (default)  
- **Typed CRDs** (optional)

Both modes share the same reconciler, same health API, same metrics, and same
dependency model.

---

## Dynamic CRDs

Dynamic CRDs use:

- dynamic client  
- unstructured.Unstructured  
- no compiled Go types  
- no scheme registration  
- no code generation  

### Informer path

NewDynamicListerWatcher → ForListerWatcher → SharedIndexInformer


The informer stores `*unstructured.Unstructured`.

### Best for

- rapid prototyping  
- YAML‑only operators  
- CRDs without Go types  
- zero‑code operators  

---

## Typed CRDs

Typed CRDs use:

- typed REST client  
- compiled Go structs  
- scheme registration  
- code generation (`ork generate runtime`)  

### Informer path

ClientProvider.Register → infFactory.For → SharedIndexInformer


The informer stores typed objects.

### Best for

- complex CRDs  
- type‑safe hooks  
- external API calls  
- advanced reconcile logic  

---

## Runtime Equivalence

After the informer stage, both modes are identical:

- same GenericReconciler  
- same hooks  
- same template engine  
- same metrics  
- same health API  
- same dependency model  
