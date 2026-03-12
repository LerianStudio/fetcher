package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCertificates(t *testing.T) {
	opts := DefaultGenerateOptions()
	bundle, err := GenerateCertificates(opts)
	require.NoError(t, err)

	t.Run("CA certificate is valid", func(t *testing.T) {
		cert, err := x509.ParseCertificate(bundle.CACert)
		require.NoError(t, err)
		assert.True(t, cert.IsCA)
		assert.Equal(t, opts.CACommonName, cert.Subject.CommonName)
	})

	t.Run("Server certificate is valid", func(t *testing.T) {
		cert, err := x509.ParseCertificate(bundle.ServerCert)
		require.NoError(t, err)
		assert.False(t, cert.IsCA)
		assert.Equal(t, opts.ServerCommonName, cert.Subject.CommonName)
		assert.Contains(t, cert.DNSNames, "localhost")
		assert.Contains(t, cert.DNSNames, "postgres-external")
	})

	t.Run("Client certificate is valid", func(t *testing.T) {
		cert, err := x509.ParseCertificate(bundle.ClientCert)
		require.NoError(t, err)
		assert.False(t, cert.IsCA)
		assert.Equal(t, opts.ClientCommonName, cert.Subject.CommonName)
	})

	t.Run("Server certificate is signed by CA", func(t *testing.T) {
		caCert, err := x509.ParseCertificate(bundle.CACert)
		require.NoError(t, err)

		serverCert, err := x509.ParseCertificate(bundle.ServerCert)
		require.NoError(t, err)

		roots := x509.NewCertPool()
		roots.AddCert(caCert)

		_, err = serverCert.Verify(x509.VerifyOptions{
			Roots: roots,
		})
		require.NoError(t, err)
	})

	t.Run("TLS config can be created", func(t *testing.T) {
		// Create CA pool
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM([]byte(bundle.CACertPEM))
		require.True(t, ok)

		// Load server certificate
		serverCert, err := tls.X509KeyPair([]byte(bundle.ServerCertPEM), []byte(bundle.ServerKeyPEM))
		require.NoError(t, err)

		// Create TLS config
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    caCertPool,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MinVersion:   tls.VersionTLS12,
		}

		assert.NotNil(t, tlsConfig)
	})
}

func TestWriteToDir(t *testing.T) {
	opts := DefaultGenerateOptions()
	bundle, err := GenerateCertificates(opts)
	require.NoError(t, err)

	// Create temp directory
	tmpDir := filepath.Join(os.TempDir(), "fetcher-ssl-test")
	defer os.RemoveAll(tmpDir)

	err = bundle.WriteToDir(tmpDir)
	require.NoError(t, err)

	t.Run("All files exist", func(t *testing.T) {
		assert.FileExists(t, bundle.CACertPath)
		assert.FileExists(t, bundle.CAKeyPath)
		assert.FileExists(t, bundle.ServerCertPath)
		assert.FileExists(t, bundle.ServerKeyPath)
		assert.FileExists(t, bundle.ClientCertPath)
		assert.FileExists(t, bundle.ClientKeyPath)
	})

	t.Run("Files have correct permissions", func(t *testing.T) {
		// Key files should be readable only by owner
		info, err := os.Stat(bundle.CAKeyPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

		info, err = os.Stat(bundle.ServerKeyPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("Files can be loaded back", func(t *testing.T) {
		caCert, err := os.ReadFile(bundle.CACertPath)
		require.NoError(t, err)
		assert.Equal(t, bundle.CACertPEM, string(caCert))
	})
}
