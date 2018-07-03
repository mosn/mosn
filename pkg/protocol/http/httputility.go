package http

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strings"
)

func ParseQueryString(query string) types.QueryParams {

	var QueryParams = make(types.QueryParams, 10)

	queryMaps := strings.Split(query, "&")

	for _, qm := range queryMaps {
		queryMap := strings.Split(qm, "=")
		if len(queryMap) != 2 {
			log.DefaultLogger.Errorf("parse query parameters error,parameters = %s", qm)
		} else {
			QueryParams[strings.TrimSpace(queryMap[0])] = QueryParams[strings.TrimSpace(queryMap[1])]
		}
	}

	return QueryParams
}
