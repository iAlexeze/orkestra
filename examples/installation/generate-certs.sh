#!/usr/bin/env bash
# installation/generate-certs.sh
#
# Generates a self-signed certificate for local testing of Orkestra's
# admission webhook and conversion server.
#
# The certificate is valid for:
#   - orkestra.orkestra.svc
#   - orkestra.orkestra.svc.cluster.local
#   - localhost (for port-forwarding tests)
#
# The CA certificate is embedded in the webhook configurations as caBundle.
# Since this is self-signed, the cert and CA are the same file.
#
# Usage:
#   chmod +x generate-certs.sh
#   ./generate-certs.sh
#
# Output:
#   tls.crt — serving certificate (also used as caBundle)
#   tls.key — serving key
#   orkestra-tls secret applied to the cluster

set -euo pipefail

NAMESPACE="orkestra-system"
SERVICE_NAME="orkestra"
CERT_DIR="/tmp/tls"

mkdir -p "${CERT_DIR}"

echo "Generating self-signed certificate for ${SERVICE_NAME}.${NAMESPACE}.svc ..."

# Generate CA key and certificate (self-signed — same as serving cert for local test)
openssl genrsa -out "${CERT_DIR}/ca.key" 2048

openssl req -new -x509 -days 365 \
  -key "${CERT_DIR}/ca.key" \
  -out "${CERT_DIR}/ca.crt" \
  -subj "/CN=${SERVICE_NAME}-ca"

# Generate server key
openssl genrsa -out "${CERT_DIR}/tls.key" 2048

# Generate CSR with SANs
openssl req -new \
  -key "${CERT_DIR}/tls.key" \
  -out "${CERT_DIR}/tls.csr" \
  -subj "/CN=${SERVICE_NAME}.${NAMESPACE}.svc" \
  -config <(cat /etc/ssl/openssl.cnf <(printf "\n[SAN]\nsubjectAltName=DNS:%s.%s.svc,DNS:%s.%s.svc.cluster.local,DNS:localhost" \
    "${SERVICE_NAME}" "${NAMESPACE}" "${SERVICE_NAME}" "${NAMESPACE}"))

# Sign the certificate with the CA
openssl x509 -req -days 365 \
  -in "${CERT_DIR}/tls.csr" \
  -CA "${CERT_DIR}/ca.crt" \
  -CAkey "${CERT_DIR}/ca.key" \
  -CAcreateserial \
  -out "${CERT_DIR}/tls.crt" \
  -extensions SAN \
  -extfile <(printf "[SAN]\nsubjectAltName=DNS:%s.%s.svc,DNS:%s.%s.svc.cluster.local,DNS:localhost" \
    "${SERVICE_NAME}" "${NAMESPACE}" "${SERVICE_NAME}" "${NAMESPACE}")

echo "Certificate generated:"
echo "  ${CERT_DIR}/tls.crt  (serving cert + CA bundle)"
echo "  ${CERT_DIR}/tls.key  (serving key)"

# Create the Kubernetes namespace if it doesn't exist
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# Create the TLS secret
kubectl create secret tls orkestra-tls \
  --cert="${CERT_DIR}/tls.crt" \
  --key="${CERT_DIR}/tls.key" \
  --namespace="${NAMESPACE}" \
  --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "Secret 'orkestra-tls' created in namespace '${NAMESPACE}'."
echo ""
echo "The caBundle for webhook configurations will be read from TLS_CERT=/tls/tls.crt"
echo "Orkestra reads it at startup and injects it into the webhook configurations."

# Cleanup Temporary Dir
# rm -rf "${CERT_DIR}"
#