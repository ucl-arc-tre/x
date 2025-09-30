package graceful

import (
	"bytes"
	"io"
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
		"Recieved termination signal",
		"Closing server",
		"Server exited",
	}
	for _, expectedLine := range expectedLines {
		assert.Contains(t, logStream, expectedLine)
	}
	assert.Regexp(t, regexp.MustCompile(`Started.*\n.*Recieved termination signal`), logStream)
}

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}
