package types

// MockClient mocks a downstream client
type MockClient interface {
	// SyncCall sends a sync request, if returns true means request receives expected response.
	SyncCall() bool
	// AsyncCall sends a request but not wait for responses.
	AsyncCall()
	// Stats returns client metrics. If AsyncCall is called, use metrics to check results.
	Stats() ClientStatsReadOnly
	// Close closes all client connections
	Close()
}
