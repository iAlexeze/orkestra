package certmanager

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// TLS certificate generation
// ─────────────────────────────────────────────────────────────────────────────

// TLSBundle holds the generated certificate material.
type TLSBundle struct {
	CertPEM   []byte // tls.crt — signed server certificate, PEM
	KeyPEM    []byte // tls.key — server private key, PEM
	CACertPEM []byte // ca.crt  — CA certificate, PEM (for caBundle in webhooks)
}

// GenerateTLSBundle generates a self-signed CA and a server certificate signed by it.
// The server certificate has the given common name and DNS SANs.
// validFor is the certificate validity duration ("1y", "90d", etc.).
//
// Returns a TLSBundle containing PEM-encoded cert, key, and CA cert.
// All three are stored in the Secret so consumers have what they need:
//   - tls.crt + tls.key for the server
//   - ca.crt for clients that need to verify the server cert
func GenerateTLSBundle(commonName string, dnsNames []string, validFor string) (*TLSBundle, error) {
	validity := 365 * 24 * time.Hour // default: 1 year
	if validFor != "" {
		if d, err := orktypes.ParseTimeDuration(validFor); err == nil {
			validity = d
		}
	}

	// ── Step 1: Generate CA ───────────────────────────────────────────────────
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: randomSerial(),
		Subject: pkix.Name{
			CommonName:   "orkestra-ca",
			Organization: []string{"Orkestra"},
		},
		NotBefore:             time.Now().Add(-5 * time.Minute), // clock skew tolerance
		NotAfter:              time.Now().Add(validity + 24*time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("creating CA certificate: %w", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("parsing CA certificate: %w", err)
	}

	// ── Step 2: Generate server key + cert signed by the CA ───────────────────
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating server key: %w", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: randomSerial(),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Orkestra"},
		},
		DNSNames:    dnsNames,
		NotBefore:   time.Now().Add(-5 * time.Minute),
		NotAfter:    time.Now().Add(validity),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("creating server certificate: %w", err)
	}

	// ── Step 3: PEM encode everything ─────────────────────────────────────────
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	return &TLSBundle{
		CertPEM:   certPEM,
		KeyPEM:    keyPEM,
		CACertPEM: caCertPEM,
	}, nil
}

func randomSerial() *big.Int {
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return serial
}
