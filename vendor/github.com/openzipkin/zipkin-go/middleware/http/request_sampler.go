package http

import "net/http"

// RequestSamplerFunc can be implemented for client and/or server side sampling decisions that can override the existing
// upstream sampling decision. If the implementation returns nil, the existing sampling decision stays as is.
type RequestSamplerFunc func(r *http.Request) *bool

// Sample is a convenience function that returns a pointer to a boolean true. Use this for RequestSamplerFuncs when
// wanting the RequestSampler to override the sampling decision to yes.
func Sample() *bool {
	sample := true
	return &sample
}

// Discard is a convenience function that returns a pointer to a boolean false. Use this for RequestSamplerFuncs when
// wanting the RequestSampler to override the sampling decision to no.
func Discard() *bool {
	sample := false
	return &sample
}
