---
title: "Untold"
weight: 15
---

You’ve fixed **everything except the final TLS requirement**.  
This last error is *exactly* what Kubernetes gives you when:

- The certificate **has SANs**, ✔️  
- The certificate **matches the DNS name**, ✔️  
- But the certificate **is self‑signed**, ❌  
- And the CRD’s `caBundle` does **not** contain the certificate’s CA  

Here’s the key line:

```
x509: certificate signed by unknown authority
```

This means:

> Kubernetes *trusts* the DNS name now,  
> but it does **not trust the CA** that signed your certificate.

This is the final step.

---

# 🎯 What Kubernetes expects

For a conversion webhook, Kubernetes requires:

1. The certificate must have SANs matching the service name  
2. The certificate must be signed by a CA  
3. The **CA’s public certificate** must be embedded in the CRD’s `caBundle`

Right now, you generated a **self‑signed cert**, but you did **not** embed the CA into the CRD.

So Kubernetes says:

> “I see the cert, but I don’t trust the authority that signed it.”

---

# 🔥 How to fix it (final step)

You must:

### **1. Generate a CA**
```bash
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 365 \
  -out ca.crt \
  -subj "/CN=orkestra-ca"
```

### **2. Generate a CSR for the webhook cert**
Use the SAN config you already created:

```bash
openssl genrsa -out tls.key 2048
openssl req -new -key tls.key -out tls.csr -config csr.conf
```

### **3. Sign the webhook cert with the CA**
```bash
openssl x509 -req -in tls.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out tls.crt -days 365 \
  -extensions req_ext -extfile csr.conf
```

### **4. Create the Kubernetes TLS secret**
```bash
kubectl create secret tls orkestra-tls \
  --cert=tls.crt \
  --key=tls.key \
  -n orkestra
```

### **5. Base64‑encode the CA and put it in the CRD**
```bash
cat ca.crt | base64 -w0
```

Paste that into:

```yaml
conversion:
  webhook:
    clientConfig:
      caBundle: <base64-of-ca.crt>
```

### **6. Apply the CRD again**
```bash
kubectl apply -f your-crd.yaml
```

### **7. Restart Orkestra**
```bash
kubectl rollout restart deploy/orkestra-runtime -n orkestra
```

---

# 🧪 After this, your logs will show:

### Before (current):
```
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

### After (correct):
```
conversion https server listening on :8443
POST /convert 200 OK
```

And your Website CRs will finally convert.

---

# 🧠 Why this is the last blocker

You now have:

- Correct SANs  
- Correct DNS  
- Correct Service  
- Correct TLS mounting  
- Correct env vars  
- Correct RBAC  
- Correct Deployment  
- Correct webhook path  
- Correct conversion rules  

The only missing piece was:

> Kubernetes must trust the CA that signed your webhook certificate.

Once you embed the CA into the CRD, the API server will trust the webhook and conversion will work.
