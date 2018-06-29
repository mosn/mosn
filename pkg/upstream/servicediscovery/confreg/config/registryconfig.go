package config

import "time"

type RegistryConfig struct {
	//unit:s
	RPCServerPort                            int
	RegistryEndpointPort                     int
	ScheduleCompensateRegisterTaskDuration   time.Duration
	ScheduleRefreshConfregServerTaskDuration time.Duration
	ConnectRetryDuration                     time.Duration
	RegisterTimeout                          time.Duration
	WaitReceivedDataTimeout                  time.Duration
}

var DefaultRegistryConfig = &RegistryConfig{
	RPCServerPort:                            12201,
	RegistryEndpointPort:                     9603,
	ScheduleCompensateRegisterTaskDuration:   60 * time.Second,
	ScheduleRefreshConfregServerTaskDuration: 60 * time.Second,
	RegisterTimeout:                          3 * time.Second,
	ConnectRetryDuration:                     5 * time.Second,
	WaitReceivedDataTimeout:                  3 * time.Second,
}