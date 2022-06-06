package main

import (
	"encoding/json"
	"errors"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"google.golang.org/grpc/credentials"
	"mosn.io/pkg/log"
)

const (
	LogReporter        string = "log"
	GRPCReporter       string = "gRPC"
	DefaultServiceName string = "mosn"
)

type SkyWalkingTraceConfig struct {
	Reporter         string                   `json:"reporter"`
	BackendService   string                   `json:"backend_service"`
	ServiceName      string                   `json:"service_name"`
	MaxSendQueueSize int                      `json:"max_send_queue_size"`
	Authentication   string                   `json:"authentication"`
	TLS              SkyWalkingTraceTLSConfig `json:"tls"`
}

type SkyWalkingTraceTLSConfig struct {
	CertFile           string `json:"cert_file"`
	ServerNameOverride string `json:"server_name_override"`
}

var (
	ReporterCfgErr       = errors.New("SkyWalking tracer support only log and gRPC reporter")
	BackendServiceCfgErr = errors.New("SkyWalking tracer must configure the backend_service")
)

func newGO2SkyTracer(config map[string]interface{}) (t *go2sky.Tracer, err error) {
	cfg, err := parseAndVerifySkyTracerConfig(config)
	if err != nil {
		return nil, err
	}

	var r go2sky.Reporter
	if cfg.Reporter == LogReporter {
		r, err = reporter.NewLogReporter()
		if err != nil {
			return nil, err
		}
	} else if cfg.Reporter == GRPCReporter {
		// opts
		var opts []reporter.GRPCReporterOption
		// max send queue size
		if cfg.MaxSendQueueSize > 0 {
			opts = append(opts, reporter.WithMaxSendQueueSize(cfg.MaxSendQueueSize))
		}
		// auth
		if cfg.Authentication != "" {
			opts = append(opts, reporter.WithAuthentication(cfg.Authentication))
		}
		// tls
		if cfg.TLS.CertFile != "" {
			creds, err := credentials.NewClientTLSFromFile(cfg.TLS.CertFile, cfg.TLS.ServerNameOverride)
			if err != nil {
				return nil, err
			}
			opts = append(opts, reporter.WithTransportCredentials(creds))
		}

		r, err = reporter.NewGRPCReporter(cfg.BackendService, opts...)
		if err != nil {
			return nil, err
		}
	}

	t, err = go2sky.NewTracer(cfg.ServiceName, go2sky.WithReporter(r))
	if err != nil {
		return nil, err
	}
	return
}

func parseAndVerifySkyTracerConfig(cfg map[string]interface{}) (config SkyWalkingTraceConfig, err error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return config, err
	}
	log.DefaultLogger.Debugf("[SkyWalking] [tracer] tracer config: %v", string(data))

	// set default value
	config.Reporter = LogReporter
	config.ServiceName = DefaultServiceName

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	if config.Reporter != LogReporter && config.Reporter != GRPCReporter {
		return config, ReporterCfgErr
	}

	if config.Reporter == GRPCReporter && config.BackendService == "" {
		return config, BackendServiceCfgErr
	}
	return config, nil
}
