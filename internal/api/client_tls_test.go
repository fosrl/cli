package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

func TestCreateHTTPClientWithTLSClientCert(t *testing.T) {
	pemPath, p12Path := writeTestClientCerts(t)

	t.Run("pem", func(t *testing.T) {
		client, err := createHTTPClient(time.Second, pemPath)
		if err != nil {
			t.Fatalf("createHTTPClient() error = %v", err)
		}
		assertTLSClientCert(t, client, "pangolin-client")
	})

	t.Run("pkcs12", func(t *testing.T) {
		client, err := createHTTPClient(time.Second, p12Path)
		if err != nil {
			t.Fatalf("createHTTPClient() error = %v", err)
		}
		assertTLSClientCert(t, client, "pangolin-client")
	})
}

func assertTLSClientCert(t *testing.T, client *http.Client, wantCommonName string) {
	t.Helper()

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("client.Transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.TLSClientConfig == nil {
		t.Fatalf("transport.TLSClientConfig = nil")
	}
	if got := len(transport.TLSClientConfig.Certificates); got != 1 {
		t.Fatalf("len(Certificates) = %d, want 1", got)
	}
	leaf := transport.TLSClientConfig.Certificates[0].Leaf
	if leaf == nil {
		t.Fatalf("certificate Leaf = nil")
	}
	if got := leaf.Subject.CommonName; got != wantCommonName {
		t.Fatalf("certificate CommonName = %q, want %q", got, wantCommonName)
	}
}

func writeTestClientCerts(t *testing.T) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "pangolin-client",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}

	dir := t.TempDir()

	pemPath := filepath.Join(dir, "client.pem")
	pemFile, err := os.Create(pemPath)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", pemPath, err)
	}
	if err := pem.Encode(pemFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("encode certificate PEM: %v", err)
	}
	if err := pem.Encode(pemFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		t.Fatalf("encode key PEM: %v", err)
	}
	if err := pemFile.Close(); err != nil {
		t.Fatalf("close PEM file: %v", err)
	}

	p12Path := filepath.Join(dir, "client.p12")
	p12Data, err := pkcs12.Encode(rand.Reader, key, cert, nil, "")
	if err != nil {
		t.Fatalf("pkcs12.Encode() error = %v", err)
	}
	if err := os.WriteFile(p12Path, p12Data, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", p12Path, err)
	}

	return pemPath, p12Path
}
