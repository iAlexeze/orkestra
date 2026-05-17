# Run Pipeline

`runner.Run` executes these steps in order. Each step must succeed before the next begins.

```
1. Capture original kubectl context  (restored via defer on any exit)
2. Ensure cluster                    (create kind cluster, or use --cluster)
3. Check dependencies                (kubectl, helm, kind available)
4. Apply CRD                         (from spec.crd or katalog crdFile entries)
5. Generate bundle                   (ork generate bundle → resolves crdFile inline)
6. Apply bundle                      (kubectl apply — RBAC, ServiceAccounts, ConfigMap)
7. Apply setup files                 (spec.setup, in order)
8. Install Orkestra                  (helm install, skipped if already installed)
9. Wait for Orkestra ready           (health check)
10. Run expectations                 (cr-applied / cr-deleted, in order)
11. Print results
12. Delete cluster                   (unless --keep-cluster or --cluster was used)
13. Restore kubectl context          (defer — always runs)
```

## Bundle generation

The runner resolves `crdFile` references in the Katalog before generating the bundle. This embeds the CRD type information directly into the ConfigMap so the in-cluster Orkestra runtime doesn't need access to local files.

## Orkestra install

If Orkestra is already installed in the cluster (`doctor.OrkestraInstalled()` returns true), the install step is skipped. This makes `--keep-cluster` + `reuse: true` fast for local iteration.

## Context restore

The original `kubectl` context is captured at the start of `Run` and restored via `defer` — so it's always restored regardless of whether the test passes, fails, or panics. This prevents the kind cluster context from stranding your terminal after the cluster is deleted.
