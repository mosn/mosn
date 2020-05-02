package config

type FaultToleranceRuleJson struct {
	Enabled               bool     `json:"enabled"`
	ExceptionTypes        []string `json:"exceptionTypes"`
	TimeWindow            int64    `json:"timeWindow"`
	LeastWindowCount      int64    `json:"leastWindowCount"`
	ExceptionRateMultiple float64  `json:"exceptionRateMultiple"`
	MaxIpCount            int64    `json:"maxIpCount"`
	MaxIpRatio            float64  `json:"maxIpRatio"`
	RecoverTime           int64    `json:"recoverTime"`
}
