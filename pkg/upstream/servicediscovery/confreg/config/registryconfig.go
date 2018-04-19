package config

import "time"

type RegistryConfig struct {
    //unit:s
    ScheduleRegisterTaskDuration time.Duration
    RegisterTimeout              time.Duration
    ConnectRetryDuration         time.Duration
}

func DefaultRegistryConfig() *RegistryConfig {
    return &RegistryConfig{
        ScheduleRegisterTaskDuration: 60 * time.Second,
        RegisterTimeout:              30 * time.Second,
        ConnectRetryDuration:         5 * time.Second,
    }
}
