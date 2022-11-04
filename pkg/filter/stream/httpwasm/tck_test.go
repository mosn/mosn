package wasm_test

import (
	"testing"

	"github.com/http-wasm/http-wasm-host-go/tck"
)

func TestTCK(t *testing.T) {
	// Initialize the TCK backend which is used to verify downstream calls.
	backend := tck.StartBackend("")
	defer backend.Close()

	// Start Mosn with the TCK Guest, which serves responses or dispatches to
	// the backend.
	mosn := startMosn(t, backend.Listener.Addr().String(), tck.GuestWASM)
	defer mosn.Close()

	// Run tests, issuing HTTP requests to server.
	tck.Run(t, mosn.url)
}
