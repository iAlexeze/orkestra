# 🚀 Start Here — The Orkestra Onboarding Guide

Welcome to Orkestra — the operator framework where **your CRD is enough**.

This guide gives you the mental model, the workflow, and your first operator in minutes.

---

# 1. What Orkestra Is

Orkestra is a **runtime**, not a framework.

You don’t write:

- Go  
- Python  
- JavaScript  
- Reconcile loops  
- Controllers  
- Webhooks  
- CRD structs  
- Boilerplate  

You write:

- **A CRD** (the schema)  
- **A Katalog** (the behavior)  

Orkestra does the rest.

---

# 2. The Mental Model

```
CRD → Katalog → Orkestra → Kubernetes
```

- **CRD** defines *what* your resource is  
- **Katalog** defines *how* it should behave  
- **Orkestra** reconciles it  
- **Kubernetes** stores it  

This is the simplest operator model ever created.

---

# 3. Install Orkestra

```bash
brew tap iAlexeze/tap
brew install ork
```

Or:

```bash
curl -sSL https://raw.githubusercontent.com/iAlexeze/orkestra/main/install.sh | bash
```

---

# 4. Create Your First Operator

```bash
ork init my-operator
cd my-operator
```

This gives you a ready‑to‑run Katalog.

---

# 5. Install Your CRD

```bash
kubectl apply -f examples/website/website-crd.yaml
```

---

# 6. Run Orkestra

```bash
ork run --katalog examples/website/website-katalog.yaml
```

---

# 7. Apply a CR

```bash
kubectl apply -f examples/website/website-cr.yaml
```

---

# 8. Watch Orkestra Work

```bash
ork status -w
kubectl get deployments
kubectl get services
```

---

# 9. What Just Happened?

Orkestra:

- watched your CR  
- validated it  
- mutated defaults  
- resolved templates  
- created a Deployment and Service  
- drift‑corrected them  
- exposed metrics  
- tracked health  

All from a single YAML file.

---

# 10. Next Steps

- Learn the **[Katalog](./katalog.md)**  
- Compose multiple operators with **[Komposer](./komposer.md)**  
- Explore the **[OrkestraRegistry](./orkestra-registry/orkestra-registry-vision.md)**  
- Read the philosophy: **[Your CRD is Enough](../publications/your-crd-is-enough.md)**

Welcome to the future of operators.