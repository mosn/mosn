package base

import "fmt"

type SentinelRule interface {
	fmt.Stringer

	ResourceName() string
}
