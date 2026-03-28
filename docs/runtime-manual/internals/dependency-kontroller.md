# DependencyKontroller (Internals)

The DependencyKontroller is the heart of Orkestra’s lifecycle model.

It ensures:

- CRDs start in dependency order  
- CRDs shut down in reverse order  
- missing CRDs activate automatically  
- deleted CRDs deactivate cleanly  
- dependents never start before dependencies  

---

## Ready Channels

Each CRD has a `readyCh`:

- closed when workers start  
- never closed on deactivation  
- dependents wait on it  

---

## Activation

A CRD activates when:

- informer starts  
- workers start  
- readyCh closes  
- health flips to started  

---

## Deactivation

A CRD deactivates when:

- CRD disappears  
- workers stop  
- health flips to degraded  
- informer continues running  
- retry loop marks it missing  

---

## Reactivation

When the CRD reappears:

- informer restarts cleanly  
- workers restart  
- readyCh is already closed (safe)  
- dependents become healthy again  

---

## Retry Loop

Runs forever:

- activates missing CRDs  
- deactivates deleted CRDs  
- updates health  
- exponential backoff  
