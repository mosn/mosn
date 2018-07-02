package types

import "time"

type Span interface {
	SetOperation(operation string)

	SetTag(key string, value string)

	FinishSpan()

	InjectContext()

	SpawnChild() Span
}

type Driver interface {
	start(requestHeaders map[string]string, operationName string, startTime time.Time) Span
}
