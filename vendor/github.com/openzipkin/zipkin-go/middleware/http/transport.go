// Copyright 2019 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"

	zipkin "github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
)

// ErrHandler allows instrumentations to decide how to tag errors
// based on the response status code >399 and the error from the
// Transport.RoundTrip
type ErrHandler func(sp zipkin.Span, err error, statusCode int)

func defaultErrHandler(sp zipkin.Span, err error, statusCode int) {
	if err != nil {
		zipkin.TagError.Set(sp, err.Error())
		return
	}

	statusCodeVal := strconv.FormatInt(int64(statusCode), 10)
	zipkin.TagError.Set(sp, statusCodeVal)
}

// ErrResponseReader allows instrumentations to read the error body
// and decide to obtain information to it and add it to the span i.e.
// tag the span with a more meaningful error code or with error details.
type ErrResponseReader func(sp zipkin.Span, body io.Reader)

type transport struct {
	tracer            *zipkin.Tracer
	rt                http.RoundTripper
	httpTrace         bool
	defaultTags       map[string]string
	errHandler        ErrHandler
	errResponseReader *ErrResponseReader
	logger            *log.Logger
	requestSampler    RequestSamplerFunc
	remoteEndpoint    *model.Endpoint
}

// TransportOption allows one to configure optional transport configuration.
type TransportOption func(*transport)

// RoundTripper adds the Transport RoundTripper to wrap.
func RoundTripper(rt http.RoundTripper) TransportOption {
	return func(t *transport) {
		if rt != nil {
			t.rt = rt
		}
	}
}

// TransportTags adds default Tags to inject into transport spans.
func TransportTags(tags map[string]string) TransportOption {
	return func(t *transport) {
		t.defaultTags = tags
	}
}

// TransportTrace allows one to enable Go's net/http/httptrace.
func TransportTrace(enable bool) TransportOption {
	return func(t *transport) {
		t.httpTrace = enable
	}
}

// TransportErrHandler allows to pass a custom error handler for the round trip
func TransportErrHandler(h ErrHandler) TransportOption {
	return func(t *transport) {
		t.errHandler = h
	}
}

// TransportErrResponseReader allows to pass a custom ErrResponseReader
func TransportErrResponseReader(r ErrResponseReader) TransportOption {
	return func(t *transport) {
		t.errResponseReader = &r
	}
}

// TransportRemoteEndpoint will set the remote endpoint for all spans.
func TransportRemoteEndpoint(remoteEndpoint *model.Endpoint) TransportOption {
	return func(c *transport) {
		c.remoteEndpoint = remoteEndpoint
	}
}

// TransportLogger allows to plug a logger into the transport
func TransportLogger(l *log.Logger) TransportOption {
	return func(t *transport) {
		t.logger = l
	}
}

// TransportRequestSampler allows one to set the sampling decision based on
// the details found in the http.Request. It has preference over the existing
// sampling decision contained in the context. The function returns a *bool,
// if returning nil, sampling decision is not being changed whereas returning
// something else than nil is being used as sampling decision.
func TransportRequestSampler(sampleFunc RequestSamplerFunc) TransportOption {
	return func(t *transport) {
		t.requestSampler = sampleFunc
	}
}

// NewTransport returns a new Zipkin instrumented http RoundTripper which can be
// used with a standard library http Client.
func NewTransport(tracer *zipkin.Tracer, options ...TransportOption) (http.RoundTripper, error) {
	if tracer == nil {
		return nil, ErrValidTracerRequired
	}

	t := &transport{
		tracer:     tracer,
		rt:         http.DefaultTransport,
		httpTrace:  false,
		errHandler: defaultErrHandler,
		logger:     log.New(os.Stderr, "", log.LstdFlags),
	}

	for _, option := range options {
		option(t)
	}

	return t, nil
}

// RoundTrip satisfies the RoundTripper interface.
func (t *transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	sp, _ := t.tracer.StartSpanFromContext(
		req.Context(), req.URL.Scheme+"/"+req.Method, zipkin.Kind(model.Client), zipkin.RemoteEndpoint(t.remoteEndpoint),
	)

	for k, v := range t.defaultTags {
		sp.Tag(k, v)
	}

	if t.httpTrace {
		sptr := spanTrace{
			Span: sp,
		}
		sptr.c = &httptrace.ClientTrace{
			GetConn:              sptr.getConn,
			GotConn:              sptr.gotConn,
			PutIdleConn:          sptr.putIdleConn,
			GotFirstResponseByte: sptr.gotFirstResponseByte,
			Got100Continue:       sptr.got100Continue,
			DNSStart:             sptr.dnsStart,
			DNSDone:              sptr.dnsDone,
			ConnectStart:         sptr.connectStart,
			ConnectDone:          sptr.connectDone,
			TLSHandshakeStart:    sptr.tlsHandshakeStart,
			TLSHandshakeDone:     sptr.tlsHandshakeDone,
			WroteHeaders:         sptr.wroteHeaders,
			Wait100Continue:      sptr.wait100Continue,
			WroteRequest:         sptr.wroteRequest,
		}

		req = req.WithContext(
			httptrace.WithClientTrace(req.Context(), sptr.c),
		)
	}

	zipkin.TagHTTPMethod.Set(sp, req.Method)
	zipkin.TagHTTPPath.Set(sp, req.URL.Path)

	spCtx := sp.Context()
	if t.requestSampler != nil {
		if shouldSample := t.requestSampler(req); shouldSample != nil {
			spCtx.Sampled = shouldSample
		}
	}

	_ = b3.InjectHTTP(req)(spCtx)

	res, err = t.rt.RoundTrip(req)
	if err != nil {
		t.errHandler(sp, err, 0)
		sp.Finish()
		return
	}

	if res.ContentLength > 0 {
		zipkin.TagHTTPResponseSize.Set(sp, strconv.FormatInt(res.ContentLength, 10))
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		statusCode := strconv.FormatInt(int64(res.StatusCode), 10)
		zipkin.TagHTTPStatusCode.Set(sp, statusCode)
		if res.StatusCode > 399 {
			t.errHandler(sp, nil, res.StatusCode)

			if t.errResponseReader != nil {
				sBody, err := ioutil.ReadAll(res.Body)
				if err == nil {
					res.Body.Close()
					(*t.errResponseReader)(sp, ioutil.NopCloser(bytes.NewBuffer(sBody)))
					res.Body = ioutil.NopCloser(bytes.NewBuffer(sBody))
				} else {
					t.logger.Printf("failed to read the response body in the ErrResponseReader: %v", err)
				}
			}
		}
	}
	sp.Finish()
	return
}
