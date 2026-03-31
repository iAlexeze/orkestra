# Generate TLS Certificates with CertManager

:::tip[Recommended!]
This is the recommended method for production clusters.  
cert‑manager handles certificate issuance, rotation, and CA trust automatically.
:::

:::warning
If you previously installed a self‑signed certificate manually,  
delete the old secret before switching to cert‑manager:

  ```bash
  kubectl delete secret orkestra-tls -n orkestra
  ```
:::
---

## 1. Install cert‑manager

If you don’t already have it:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

Wait for pods to become ready:

```bash
kubectl get pods -n cert-manager
```

---

## 2. Create a CA Issuer (self‑signed or external)

### Option A — Self‑signed CA (simple, good for dev)

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: orkestra-selfsigned
spec:
  selfSigned: {}
```

### Option B — Real CA (production)

If you have an internal PKI:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: orkestra-ca
spec:
  ca:
    secretName: orkestra-ca-secret
```

Where `orkestra-ca-secret` contains:

- `tls.crt` — your CA certificate  
- `tls.key` — your CA private key  

---

## 3. Create a Certificate for the Orkestra Webhook

This is the key resource.  
cert‑manager will:

- Generate the TLS keypair  
- Generate a CA bundle  
- Keep it renewed  
- Store everything in a Kubernetes secret  

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: orkestra-webhook-cert
  namespace: orkestra
spec:
  secretName: orkestra-tls
  duration: 8760h # 1 year
  renewBefore: 720h # 30 days
  issuerRef:
    name: orkestra-selfsigned   # or orkestra-ca
    kind: ClusterIssuer
  commonName: orkestra.orkestra.svc
  dnsNames:
    - orkestra
    - orkestra.orkestra
    - orkestra.orkestra.svc
    - orkestra.orkestra.svc.cluster.local
```

This automatically creates:

```
secret/orchestra-tls
  tls.crt
  tls.key
  ca.crt
```

---

## 4. Patch Your CRD to Use cert‑manager’s CA Bundle

cert‑manager stores the CA in the secret under `ca.crt`.

You can extract it like this:

```bash
kubectl get secret orkestra-tls -n orkestra -o jsonpath='{.data.ca\.crt}'
```

Paste that into your CRD:

```yaml
conversion:
  strategy: Webhook
  webhook:
    clientConfig:
      service:
        name: orkestra
        namespace: orkestra
        path: /convert
      caBundle: <base64-ca-from-secret>
```

:::note
cert‑manager **does not automatically patch CRDs**.  
You must embed the CA bundle manually or automate it with a small controller.
:::