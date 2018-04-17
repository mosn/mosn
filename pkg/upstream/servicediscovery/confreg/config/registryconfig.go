package config

import "time"

type RegistryConfig struct {
    //unit:s
    ScheduleRegisterTaskDuration time.Duration
    RegisterTimeout              time.Duration
}
