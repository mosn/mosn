package config

import "time"

type RegistryConfig struct {
    //unit:s
    RegistryEndpointPort                     int
    ScheduleCompensateRegisterTaskDuration   time.Duration
    ScheduleRefreshConfregServerTaskDuration time.Duration
    RegisterTimeout                          time.Duration
    ConnectRetryDuration                     time.Duration
}

var DefaultRegistryConfig = &RegistryConfig{
    RegistryEndpointPort:                     9603,
    ScheduleCompensateRegisterTaskDuration:   60 * time.Second,
    ScheduleRefreshConfregServerTaskDuration: 60 * time.Second,
    RegisterTimeout:                          5 * time.Second,
    ConnectRetryDuration:                     5 * time.Second,
}
