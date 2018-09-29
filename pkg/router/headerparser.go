package router

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
)

func (h *headerParser) evaluateHeaders(headers types.HeaderMap, requestInfo types.RequestInfo) {
	if h == nil {
		return
	}
	for _, toAdd := range h.headersToAdd {
		value := toAdd.headerFormatter.format(requestInfo)
		if v, ok := headers.Get(toAdd.headerName.Get()); ok && len(v) > 0 && toAdd.headerFormatter.append() {
			value = fmt.Sprintf("%s,%s", v, value)
		}
		headers.Set(toAdd.headerName.Get(), value)
	}

	for _, toRemove := range h.headersToRemove {
		headers.Del(toRemove.Get())
	}
}
