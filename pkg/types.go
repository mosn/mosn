package pkg

type Priority int

const (
	HIGH   Priority = 1
	NORMAL Priority = 0
)

type Stat struct {
	Key   string
	Type  StatType
}

type StatType string

const (
	COUNTER StatType = "COUNTER"
	GAUGE StatType = "GAUGE"
	HISTOGRAM StatType = "HISTOGRAM"
)