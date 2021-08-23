package seata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	mgrpc "mosn.io/mosn/pkg/filter/network/grpc"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

const (
	CommitRequestPath   = "tcc_commit_request_path"
	RollbackRequestPath = "tcc_rollback_request_path"
)

func init() {
	mgrpc.RegisterServerHandler("seata", NewBranchTransactionServiceServer)
}

type KeepaliveOption struct {
	EnforcementPolicy struct {
		MinTime             time.Duration `json:"minTime"`
		PermitWithoutStream bool          `json:"permitWithoutStream"`
	} `json:"enforcementPolicy"`

	ServerParameters struct {
		MaxConnectionIdle     time.Duration `json:"maxConnectionIdle"`
		MaxConnectionAge      time.Duration `json:"maxConnectionAge"`
		MaxConnectionAgeGrace time.Duration `json:"maxConnectionAgeGrace"`
		Time                  time.Duration `json:"time"`
		Timeout               time.Duration `json:"timeout"`
	} `json:"serverParameters"`
}

func getEnforcementPolicy(option *KeepaliveOption) keepalive.EnforcementPolicy {
	ep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}
	if option.EnforcementPolicy.MinTime > 0 {
		ep.MinTime = option.EnforcementPolicy.MinTime
	}
	ep.PermitWithoutStream = option.EnforcementPolicy.PermitWithoutStream
	return ep
}

func getServerParameters(option *KeepaliveOption) keepalive.ServerParameters {
	sp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second,
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               time.Second,
	}
	if option.ServerParameters.MaxConnectionIdle > 0 {
		sp.MaxConnectionIdle = option.ServerParameters.MaxConnectionIdle
	}
	if option.ServerParameters.MaxConnectionAge > 0 {
		sp.MaxConnectionAge = option.ServerParameters.MaxConnectionAge
	}
	if option.ServerParameters.MaxConnectionAgeGrace > 0 {
		sp.MaxConnectionAgeGrace = option.ServerParameters.MaxConnectionAgeGrace
	}
	if option.ServerParameters.Time > 0 {
		sp.Time = option.ServerParameters.Time
	}
	if option.ServerParameters.Timeout > 0 {
		sp.Timeout = option.ServerParameters.Timeout
	}
	return sp
}

func parseKeepaliveOption(conf json.RawMessage) (*KeepaliveOption, error) {
	data, _ := json.Marshal(conf)
	v := &KeepaliveOption{}
	err := json.Unmarshal(data, v)
	return v, err
}

func NewBranchTransactionServiceServer(conf json.RawMessage, options ...grpc.ServerOption) (mgrpc.RegisteredServer, error) {
	keepaliveOption, err := parseKeepaliveOption(conf)
	if err != nil {
		return nil, err
	}
	kaep := getEnforcementPolicy(keepaliveOption)
	kasp := getServerParameters(keepaliveOption)

	ops := append(options, grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
	s := grpc.NewServer(ops...)
	apis.RegisterBranchTransactionServiceServer(s, &BranchTransactionService{})
	return s, nil
}

// BranchTransactionService when global transaction commit, transaction coordinator callback to commit branch transaction,
// when global transaction rollback, transaction coordinator callback to rollback branch transaction.
type BranchTransactionService struct {
}

// BranchCommit commit branch transaction
func (svc *BranchTransactionService) BranchCommit(ctx context.Context, request *apis.BranchCommitRequest) (*apis.BranchCommitResponse, error) {
	requestContext := &RequestContext{
		ActionContext: make(map[string]string),
		Headers:       protocol.CommonHeader{},
		Body:          buffer.NewIoBuffer(0),
		Trailers:      protocol.CommonHeader{},
	}

	requestContext.Decode(request.ApplicationData)

	resp, err := doHttp1Request(requestContext, true)
	if err != nil {
		log.DefaultLogger.Errorf("commit failed, err: %v", err)
		return &apis.BranchCommitResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    err.Error(),
		}, nil
	}

	if resp.StatusCode() == http.StatusOK {
		return &apis.BranchCommitResponse{
			ResultCode:   apis.ResultCodeSuccess,
			XID:          request.XID,
			BranchID:     request.BranchID,
			BranchStatus: apis.PhaseTwoCommitted,
		}, nil
	}
	return &apis.BranchCommitResponse{
		ResultCode:   apis.ResultCodeSuccess,
		XID:          request.XID,
		BranchID:     request.BranchID,
		BranchStatus: apis.PhaseTwoCommitFailedRetryable,
	}, nil
}

// BranchRollback rollback branch transaction
func (svc *BranchTransactionService) BranchRollback(ctx context.Context, request *apis.BranchRollbackRequest) (*apis.BranchRollbackResponse, error) {
	requestContext := &RequestContext{
		ActionContext: make(map[string]string),
		Headers:       protocol.CommonHeader{},
		Body:          buffer.NewIoBuffer(0),
		Trailers:      protocol.CommonHeader{},
	}

	requestContext.Decode(request.ApplicationData)

	resp, err := doHttp1Request(requestContext, false)
	if err != nil {
		log.DefaultLogger.Errorf("commit failed, err: %v", err)
		return &apis.BranchRollbackResponse{
			ResultCode: apis.ResultCodeFailed,
			Message:    err.Error(),
		}, nil
	}

	if resp.StatusCode() == http.StatusOK {
		return &apis.BranchRollbackResponse{
			ResultCode:   apis.ResultCodeSuccess,
			XID:          request.XID,
			BranchID:     request.BranchID,
			BranchStatus: apis.PhaseTwoRolledBack,
		}, nil
	}
	return &apis.BranchRollbackResponse{
		ResultCode:   apis.ResultCodeSuccess,
		XID:          request.XID,
		BranchID:     request.BranchID,
		BranchStatus: apis.PhaseTwoRollbackFailedRetryable,
	}, nil
}

func doHttp1Request(requestContext *RequestContext, commit bool) (*resty.Response, error) {
	var (
		host        string
		path        string
		queryString string
	)
	host = requestContext.ActionContext[types.VarHost]
	if commit {
		path = requestContext.ActionContext[CommitRequestPath]
	} else {
		path = requestContext.ActionContext[RollbackRequestPath]
	}

	u := url.URL{
		Scheme: "http",
		Path:   path,
		Host:   host,
	}
	queryString, ok := requestContext.ActionContext[types.VarQueryString]
	if ok {
		u.RawQuery = queryString
	}

	client := resty.New()
	request := client.R()
	requestContext.Headers.Range(func(key, value string) bool {
		request.SetHeader(key, value)
		return true
	})
	request.SetBody(requestContext.Body.Bytes())
	return request.Post(u.String())
}
