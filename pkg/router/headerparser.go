package router

import (
	"fmt"

	"github.com/alipay/sofa-mosn/pkg/types"
)

func (h *headerParser) evaluateHeaders(headers map[string]string, requestInfo types.RequestInfo) {
	if h == nil {
		return
	}
	for _, toAdd := range h.headersToAdd {
		value := toAdd.headerFormatter.format(requestInfo)
		if v, ok := headers[toAdd.headerName.Get()]; ok && len(v) > 0 && toAdd.headerFormatter.append() {
			value = fmt.Sprintf("%s,%s", v, value)
		}
		headers[toAdd.headerName.Get()] = value
	}

	for _, toRemove := range h.headersToRemove {
		delete(headers, toRemove.Get())
	}
}
