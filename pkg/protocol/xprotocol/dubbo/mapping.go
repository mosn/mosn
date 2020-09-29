package dubbo

import (
	"context"
	"errors"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
	"net/http"
)

func init() {
	xprotocol.RegisterMapping(ProtocolName, &dubboStatusMapping{})
}

type dubboStatusMapping struct{}

func (m *dubboStatusMapping) MappingHeaderStatusCode(ctx context.Context, headers types.HeaderMap) (int, error) {
	cmd, ok := headers.(xprotocol.XRespFrame)
	if !ok {
		return 0, errors.New("no response status in headers")
	}
	code := uint16(cmd.GetStatusCode())
	switch code {
	case 20:
		return http.StatusOK, nil
	case 30, 31:
		return http.StatusGatewayTimeout, nil
	case 40:
		return http.StatusGatewayTimeout, nil
	case 60:
		return http.StatusBadGateway, nil
	case 100:
		return http.StatusInsufficientStorage, nil
	default:
		return http.StatusInternalServerError, nil
	}
}
