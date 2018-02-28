package ratelimit

type LimitStatus string

const (
	OK LimitStatus = "OK"
	Error LimitStatus = "Error"
	OverLimit LimitStatus = "OverLimit"
)

type DescriptorEntry struct {
	Key string
	Value string
}

type Descriptor struct {
	entries []DescriptorEntry
}