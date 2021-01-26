package types

// MockServer mocks a upstream server for mosn test
type MockServer interface {
	// Start runs a mock server
	Start()
	// Close stops a mock server
	Stop()
	// Stats returns a ServerStats
	Stats() ServerStatsReadOnly
}
