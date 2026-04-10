package graceful

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestServeLogStream(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	log.Logger = zerolog.New(logBuffer)

	server := http.Server{
		Handler: http.NewServeMux(),
		Addr:    "127.0.0.1:8000",
	}
	go Serve(&server, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond) // some startup time

	process := must(os.FindProcess(os.Getpid()))
	err := process.Signal(syscall.SIGINT)
	assert.NoError(t, err)
	time.Sleep(20 * time.Millisecond) // some shutdown time > shutdown duration

	logStream := string(must(io.ReadAll(logBuffer)))
	expectedLines := []string{
		"Started HTTP server",
		"Received termination signal",
		"Closing server",
		"Server exited",
	}
	for _, expectedLine := range expectedLines {
		assert.Contains(t, logStream, expectedLine)
	}
	assert.Regexp(t, regexp.MustCompile(`Started.*\n.*Received termination signal`), logStream)
}

func TestServeTLSLogStream(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	log.Logger = zerolog.New(logBuffer)

	key := must(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
	}
	certDER := must(x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key))
	keyDER := must(x509.MarshalECPrivateKey(key))
	tlsCert := must(tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	))

	server := http.Server{
		Handler:   http.NewServeMux(),
		Addr:      "127.0.0.1:8443",
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{tlsCert}},
	}
	go ServeTLS(&server, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond) // some startup time

	process := must(os.FindProcess(os.Getpid()))
	err := process.Signal(syscall.SIGINT)
	assert.NoError(t, err)
	time.Sleep(20 * time.Millisecond) // some shutdown time > shutdown duration

	logStream := string(must(io.ReadAll(logBuffer)))
	expectedLines := []string{
		"Started HTTPS server",
		"Received termination signal",
		"Closing server",
		"Server exited",
	}
	for _, expectedLine := range expectedLines {
		assert.Contains(t, logStream, expectedLine)
	}
	assert.Regexp(t, regexp.MustCompile(`Started.*\n.*Received termination signal`), logStream)
}

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}
