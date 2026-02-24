// Package ssl provides certificate generation utilities for integration tests.
package ssl

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertificateBundle contains all certificates and keys for SSL testing.
type CertificateBundle struct {
	// CA certificate and key
	CACert    []byte
	CAKey     []byte
	CACertPEM string
	CAKeyPEM  string

	// Server certificate and key
	ServerCert    []byte
	ServerKey     []byte
	ServerCertPEM string
	ServerKeyPEM  string

	// Client certificate and key (optional)
	ClientCert    []byte
	ClientKey     []byte
	ClientCertPEM string
	ClientKeyPEM  string

	// Paths to written files (populated after WriteToDir)
	CACertPath     string
	CAKeyPath      string
	ServerCertPath string
	ServerKeyPath  string
	ClientCertPath string
	ClientKeyPath  string
}

// GenerateOptions configures certificate generation.
type GenerateOptions struct {
	// CommonName for the CA certificate
	CACommonName string

	// CommonName for the server certificate
	ServerCommonName string

	// DNS names for the server certificate (SANs)
	ServerDNSNames []string

	// IP addresses for the server certificate (SANs)
	ServerIPAddresses []net.IP

	// Whether to generate client certificates
	GenerateClientCert bool

	// CommonName for the client certificate
	ClientCommonName string

	// Validity duration for certificates
	ValidityDuration time.Duration
}

// DefaultGenerateOptions returns sensible defaults for test certificates.
func DefaultGenerateOptions() GenerateOptions {
	return GenerateOptions{
		CACommonName:       "Fetcher Test CA",
		ServerCommonName:   "localhost",
		ServerDNSNames:     []string{"localhost", "postgres-external", "mysql-external", "mssql-external", "oracle-external", "fetcher-mongodb-external"},
		ServerIPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		GenerateClientCert: true,
		ClientCommonName:   "fetcher-test-client",
		ValidityDuration:   24 * time.Hour * 365, // 1 year
	}
}

// GenerateCertificates creates a complete certificate bundle for testing.
func GenerateCertificates(opts GenerateOptions) (*CertificateBundle, error) {
	bundle := &CertificateBundle{}

	// Generate CA
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Fetcher Test"},
			CommonName:   opts.CACommonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(opts.ValidityDuration),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	bundle.CACert = caCertDER
	bundle.CAKey = x509.MarshalPKCS1PrivateKey(caKey)

	caCertPEM, err := pemEncode("CERTIFICATE", caCertDER)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CA certificate PEM: %w", err)
	}

	bundle.CACertPEM = string(caCertPEM)

	caKeyPEM, err := pemEncode("RSA PRIVATE KEY", bundle.CAKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CA key PEM: %w", err)
	}

	bundle.CAKeyPEM = string(caKeyPEM)

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Generate Server Certificate
	serverKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %w", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Fetcher Test"},
			CommonName:   opts.ServerCommonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(opts.ValidityDuration),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              opts.ServerDNSNames,
		IPAddresses:           opts.ServerIPAddresses,
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create server certificate: %w", err)
	}

	bundle.ServerCert = serverCertDER
	bundle.ServerKey = x509.MarshalPKCS1PrivateKey(serverKey)

	serverCertPEM, err := pemEncode("CERTIFICATE", serverCertDER)
	if err != nil {
		return nil, fmt.Errorf("failed to encode server certificate PEM: %w", err)
	}

	bundle.ServerCertPEM = string(serverCertPEM)

	serverKeyPEM, err := pemEncode("RSA PRIVATE KEY", bundle.ServerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode server key PEM: %w", err)
	}

	bundle.ServerKeyPEM = string(serverKeyPEM)

	// Generate Client Certificate (optional)
	if opts.GenerateClientCert {
		clientKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, fmt.Errorf("failed to generate client key: %w", err)
		}

		clientTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(3),
			Subject: pkix.Name{
				Organization: []string{"Fetcher Test"},
				CommonName:   opts.ClientCommonName,
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().Add(opts.ValidityDuration),
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			BasicConstraintsValid: true,
		}

		clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create client certificate: %w", err)
		}

		bundle.ClientCert = clientCertDER
		bundle.ClientKey = x509.MarshalPKCS1PrivateKey(clientKey)

		clientCertPEM, err := pemEncode("CERTIFICATE", clientCertDER)
		if err != nil {
			return nil, fmt.Errorf("failed to encode client certificate PEM: %w", err)
		}

		bundle.ClientCertPEM = string(clientCertPEM)

		clientKeyPEM, err := pemEncode("RSA PRIVATE KEY", bundle.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encode client key PEM: %w", err)
		}

		bundle.ClientKeyPEM = string(clientKeyPEM)
	}

	return bundle, nil
}

// WriteToDir writes all certificates to the specified directory.
func (b *CertificateBundle) WriteToDir(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write CA certificate
	b.CACertPath = filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(b.CACertPath, []byte(b.CACertPEM), 0600); err != nil {
		return fmt.Errorf("failed to write CA cert: %w", err)
	}

	// Write CA key
	b.CAKeyPath = filepath.Join(dir, "ca.key")
	if err := os.WriteFile(b.CAKeyPath, []byte(b.CAKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to write CA key: %w", err)
	}

	// Write server certificate
	b.ServerCertPath = filepath.Join(dir, "server.crt")
	if err := os.WriteFile(b.ServerCertPath, []byte(b.ServerCertPEM), 0600); err != nil {
		return fmt.Errorf("failed to write server cert: %w", err)
	}

	// Write server key
	b.ServerKeyPath = filepath.Join(dir, "server.key")
	if err := os.WriteFile(b.ServerKeyPath, []byte(b.ServerKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to write server key: %w", err)
	}

	// Write client certificate (if generated)
	if len(b.ClientCertPEM) > 0 {
		b.ClientCertPath = filepath.Join(dir, "client.crt")
		if err := os.WriteFile(b.ClientCertPath, []byte(b.ClientCertPEM), 0600); err != nil {
			return fmt.Errorf("failed to write client cert: %w", err)
		}

		b.ClientKeyPath = filepath.Join(dir, "client.key")
		if err := os.WriteFile(b.ClientKeyPath, []byte(b.ClientKeyPEM), 0600); err != nil {
			return fmt.Errorf("failed to write client key: %w", err)
		}
	}

	return nil
}

// pemEncode encodes data to PEM format.
func pemEncode(blockType string, data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
