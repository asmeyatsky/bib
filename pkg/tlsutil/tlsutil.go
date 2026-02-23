// Package tlsutil provides helpers for loading TLS credentials used by gRPC
// servers and clients in the Bank-in-a-Box platform.
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc/credentials"
)

// ServerTLSConfig loads TLS credentials for a gRPC server from cert and key files.
func ServerTLSConfig(certFile, keyFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("tlsutil: load server key pair: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsCfg), nil
}

// ClientTLSConfig loads TLS credentials for a gRPC client.
// If caFile is provided, it is used as the root CA; otherwise the system CA pool is used.
// Set insecureSkipVerify to true only for development/testing.
func ClientTLSConfig(caFile string, insecureSkipVerify bool) (credentials.TransportCredentials, error) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // intentional for dev use
	}

	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("tlsutil: read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("tlsutil: failed to parse CA certificate from %s", caFile)
		}
		tlsCfg.RootCAs = pool
	}

	return credentials.NewTLS(tlsCfg), nil
}

// GenerateSelfSignedCert generates a self-signed CA and a server certificate
// for the given hosts. Files are written to outDir:
//
//	ca.pem, ca-key.pem        – CA certificate and key
//	server.pem, server-key.pem – server certificate and key signed by the CA
func GenerateSelfSignedCert(hosts []string, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("tlsutil: mkdir %s: %w", outDir, err)
	}

	// --- CA ---
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("tlsutil: generate CA key: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"BIB Dev CA"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("tlsutil: create CA cert: %w", err)
	}

	if err := writePEM(filepath.Join(outDir, "ca.pem"), "CERTIFICATE", caDER); err != nil {
		return err
	}
	caKeyBytes, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return fmt.Errorf("tlsutil: marshal CA key: %w", err)
	}
	if err := writePEM(filepath.Join(outDir, "ca-key.pem"), "EC PRIVATE KEY", caKeyBytes); err != nil {
		return err
	}

	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return fmt.Errorf("tlsutil: parse CA cert: %w", err)
	}

	// --- Server cert ---
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("tlsutil: generate server key: %w", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{Organization: []string{"BIB Dev"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			serverTemplate.IPAddresses = append(serverTemplate.IPAddresses, ip)
		} else {
			serverTemplate.DNSNames = append(serverTemplate.DNSNames, h)
		}
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("tlsutil: create server cert: %w", err)
	}

	if err := writePEM(filepath.Join(outDir, "server.pem"), "CERTIFICATE", serverDER); err != nil {
		return err
	}
	serverKeyBytes, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return fmt.Errorf("tlsutil: marshal server key: %w", err)
	}
	return writePEM(filepath.Join(outDir, "server-key.pem"), "EC PRIVATE KEY", serverKeyBytes)
}

func writePEM(path, blockType string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("tlsutil: write %s: %w", path, err)
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}
