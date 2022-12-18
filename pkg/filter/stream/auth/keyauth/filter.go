package keyauth

import (
	"context"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/variable"

	"mosn.io/mosn/pkg/filter/stream/auth"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

type filter struct {
	rules   []*MatcherVerifierPair
	handler api.StreamReceiverFilterHandler
}

func newAuthFilter(rules []*MatcherVerifierPair) *filter {
	return &filter{
		rules: rules,
	}
}

func (f *filter) OnReceive(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("call auth filter")
	}

	requestArg, err := variable.GetString(ctx, types.VarHttpRequestArg)
	if err != nil {
		log.DefaultLogger.Errorf("[auth filter] get query parameter: %v", err)
		return api.StreamFilterContinue
	}

	requestPath, err := variable.GetString(ctx, types.VarHttpRequestPath)
	if err != nil {
		log.DefaultLogger.Errorf("[auth filter] get path: %v", err)
		return api.StreamFilterStop
	}

	var verifier auth.Verifier
	for _, rulePair := range f.rules {
		if rulePair.Matcher.Matches(headers, requestPath) {
			verifier = rulePair.Verifier
			break
		}
	}

	if accept := verifier.Verify(headers, requestArg); !accept {
		f.handler.SendHijackReply(401, headers)
		return api.StreamFilterStop
	}

	return api.StreamFilterContinue
}

func (f *filter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	f.handler = handler
}

func (f *filter) OnDestroy() {}
