```yaml
source:
  helm:
    - repo: oci://ghcr.io/orkspace/charts       # Without the oci, it does not find it
      chart: orkestra-katalog-example
      version: 0.1.0
```

```
downloading repo index from "oci://ghcr.io/orkspace/charts":
failed to perform "FetchReference" on source: invalid reference
```

This error tells us something very specific:

# 🔍 **Helm OCI registries do NOT have an index.yaml**

A classic Helm repository (HTTP-based) has:

```
index.yaml
charts/
```

But an **OCI registry does not**.  
There is **no index**, no listing, no metadata endpoint.

So when Komposer tries to:

```
download repo index from "oci://ghcr.io/orkspace/charts"
```

Helm’s OCI backend responds:

```
invalid reference
```

Because **there is no index to fetch**.

This is expected behavior.

---

# ✅ **Why Helm CLI works but Komposer fails**

When you run:

```
helm install kat oci://ghcr.io/orkspace/charts/orkestra-katalog-example --version 0.1.0
```

Helm does **not** fetch an index.  
It pulls the chart directly by digest or tag.

But your Komposer loader is still doing:

```
helm repo update
helm repo index
helm repo add
```

or something equivalent.

That logic is **only valid for HTTP Helm repos**, not OCI.

---

# 🎯 **The fix: Komposer must NOT fetch an index for OCI repos**

Your loader must branch:

## **Case 1 — repo starts with `oci://`**
- Do **not** fetch an index
- Do **not** run `helm repo add`
- Do **not** run `helm repo update`
- Instead, pull directly:

```
helm pull oci://<repo>/<chart> --version <version>
```

## **Case 2 — repo is HTTP/HTTPS**
- Use classic Helm repo logic
- Fetch index.yaml
- Resolve chart versions

---

# 🧠 **Correct Komposer logic (pseudo‑code)**

```go
if strings.HasPrefix(repo, "oci://") {
    // OCI registry
    cmd := exec.Command("helm", "pull",
        fmt.Sprintf("%s/%s", repo, chart),
        "--version", version,
        "--untar", "--untardir", tempDir,
    )
    run(cmd)
} else {
    // Classic Helm repo
    helm repo add ...
    helm repo update ...
    helm pull repo/chart --version version ...
}
```

This is the correct, Helm‑compatible behavior.

---

# 🧪 **How to confirm the fix**

After updating Komposer:

```
helm:
  - repo: oci://ghcr.io/orkspace/charts
    chart: orkestra-katalog-example
    version: 0.1.0
```

Then run:

```
ork template --katalog komposer.yaml
```

You should see:

- No “FetchReference” error  
- No “invalid reference”  
- Chart pulled successfully  
- Katalog merged correctly  

---

# 🧩 **Why this matters**

OCI registries are *not* Helm repos.  
They are just content-addressable stores.  
There is no index, no listing, no metadata.

So Komposer must treat them differently.