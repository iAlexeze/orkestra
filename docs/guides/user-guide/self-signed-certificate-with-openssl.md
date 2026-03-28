# Using OpenSSL to Generate TLS Certificates for Orkestra

Orkestra’s conversion webhook requires TLS.  
This guide walks you through generating a **self‑signed certificate** suitable for development and testing.

> !!! warning
> **Self‑signed certificates are strongly discouraged in production.** 

> They are not trusted by default, cannot be revoked, and require manual CA distribution.  

> For production clusters, use:

> - A real Certificate Authority (public or private)

> - [cert‑manager](./self-signed-certificate-with-cert-manager.md) with a proper Issuer

> - Your organization’s PKI

---

## Overview

Kubernetes requires three things for a conversion webhook:

1. A TLS certificate whose **SANs match the webhook service name**  
2. A **CA certificate** that signed the TLS certificate  
3. The **CA certificate embedded in the CRD’s `caBundle`**

This guide generates:

- A CA (`ca.crt`, `ca.key`)
- A TLS certificate (`tls.crt`, `tls.key`)
- A Kubernetes secret (`orkestra-tls`)
- A base64‑encoded CA bundle for your CRD

---

## 1. Create a Certificate Signing Request (CSR) Config

Create a file named **csr.conf**:

```ini
[ req ]
default_bits       = 2048
prompt             = no
default_md         = sha256
req_extensions     = req_ext
distinguished_name = dn

[ dn ]
CN = orkestra.orkestra.svc

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = orkestra
DNS.2 = orkestra.orkestra
DNS.3 = orkestra.orkestra.svc
DNS.4 = orkestra.orkestra.svc.cluster.local
```

> !!! tip 
> These SANs must match the service name defined in your CRD’s webhook configuration.

---

## 2. Generate a Certificate Authority (CA)

```bash
openssl genrsa -out ca.key 2048

openssl req -x509 -new -nodes \
  -key ca.key \
  -sha256 -days 365 \
  -out ca.crt \
  -subj "/CN=orkestra-ca"
```

> !!! note
> This CA will be used only to sign the webhook certificate.  
> Kubernetes will trust this CA because you will embed it in the CRD.

---

## 3. Generate the Webhook TLS Key and CSR

```bash
openssl genrsa -out tls.key 2048

openssl req -new \
  -key tls.key \
  -out tls.csr \
  -config csr.conf
```

---

## 4. Sign the Certificate Using the CA

```bash
openssl x509 -req \
  -in tls.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out tls.crt \
  -days 365 \
  -extensions req_ext -extfile csr.conf
```

This produces:

- `tls.crt` — certificate for the webhook server  
- `tls.key` — private key  
- `ca.crt` — CA certificate to embed in the CRD  

---

## 5. Create the Kubernetes TLS Secret

```bash
kubectl create secret tls orkestra-tls \
  --cert=tls.crt \
  --key=tls.key \
  -n orkestra
```

> !!! tip 
> Orkestra mounts this secret at `/tls/tls.crt` and `/tls/tls.key`.

---

## 6. Embed the CA in the CRD

Base64‑encode the CA:

```bash
cat ca.crt | base64 -w0
```

Insert the output into your CRD:

```yaml
conversion:
  strategy: Webhook
  webhook:
    clientConfig:
      service:
        name: orkestra
        namespace: orkestra
        path: /convert
      caBundle: <BASE64_CA_CERT>
```

> !!! warning
> If the CA does not match the certificate used by the webhook server,  
> Kubernetes will reject all conversion requests with:
>
> `x509: certificate signed by unknown authority`
