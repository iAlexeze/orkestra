#!/usr/bin/env bash
set -euo pipefail

echo "=== Orkestra Self-Signed Certificate Generator ==="
echo ""

read -rp "Enter Orkestra service name (default: orkestra): " SERVICE
SERVICE=${SERVICE:-orkestra}

read -rp "Enter Orkestra namespace (default: orkestra): " NAMESPACE
NAMESPACE=${NAMESPACE:-orkestra}

OUT_DIR="orkestra-certs"
CSR_CONF="$OUT_DIR/csr.conf"

mkdir -p "$OUT_DIR"

echo "Generating certificates for service: $SERVICE.$NAMESPACE.svc"
echo "Output directory: $OUT_DIR"
echo ""

# 1. Create CSR config
cat > "$CSR_CONF" <<EOF
[ req ]
default_bits       = 2048
prompt             = no
default_md         = sha256
req_extensions     = req_ext
distinguished_name = dn

[ dn ]
CN = ${SERVICE}.${NAMESPACE}.svc

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
DNS.4 = ${SERVICE}.${NAMESPACE}.svc.cluster.local
EOF

# 2. Generate CA
openssl genrsa -out "$OUT_DIR/ca.key" 2048 >/dev/null 2>&1
openssl req -x509 -new -nodes \
  -key "$OUT_DIR/ca.key" \
  -sha256 -days 365 \
  -out "$OUT_DIR/ca.crt" \
  -subj "/CN=orkestra-ca" >/dev/null 2>&1

# 3. Generate TLS key + CSR
openssl genrsa -out "$OUT_DIR/tls.key" 2048 >/dev/null 2>&1
openssl req -new \
  -key "$OUT_DIR/tls.key" \
  -out "$OUT_DIR/tls.csr" \
  -config "$CSR_CONF" >/dev/null 2>&1

# 4. Sign TLS cert with CA
openssl x509 -req \
  -in "$OUT_DIR/tls.csr" \
  -CA "$OUT_DIR/ca.crt" -CAkey "$OUT_DIR/ca.key" -CAcreateserial \
  -out "$OUT_DIR/tls.crt" \
  -days 365 \
  -extensions req_ext -extfile "$CSR_CONF" >/dev/null 2>&1

# 5. Output base64 CA bundle
base64 -w0 "$OUT_DIR/ca.crt" > "$OUT_DIR/caBundle.txt"

echo "Certificates generated successfully."
echo ""
echo "Files created in: $OUT_DIR/"
echo "  - ca.crt"
echo "  - ca.key"
echo "  - tls.crt"
echo "  - tls.key"
echo "  - tls.csr"
echo "  - csr.conf"
echo "  - caBundle.txt (base64 CA for CRD)"
echo ""
echo "Add the contents of caBundle.txt to your CRD's conversion webhook:"
echo ""
echo "conversion:"
echo "  webhook:"
echo "    clientConfig:"
echo "      caBundle: <contents of caBundle.txt>"
echo ""
echo "Done."
