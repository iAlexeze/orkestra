# Informer Factory (Internals)

The Informer Factory manages all CRD informers in Orkestra.

---

## Responsibilities

- Create one informer per CRD  
- Start informers only when CRDs exist  
- Track missing CRDs  
- Allow activation/deactivation  
- Feed events into the shared workqueue  

---

## Missing CRDs

If a CRD does not exist at startup:

- informer is created  
- informer is **not** started  
- entry is added to `missing` map  
- retryMissingCRDs will activate it later  

---

## Activation

When a missing CRD appears:

- retry loop detects it  
- informer.Run() is started  
- workers are started  
- readyCh is closed  
- dependents unblock  

---

## Deactivation

When a CRD is deleted:

- workers stop  
- health degrades  
- informer continues running (reflector errors expected)  
- retry loop will reactivate when CRD reappears  

---

## Reflector Errors

Reflector logs like:

```bash
Failed to list ... the server could not find the requested resource
```

are **expected** when CRDs disappear.

They are not errors — they are the signal that drives deactivation.
