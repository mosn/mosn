package main

import (
	"fmt"
	"strings"

	"mosn.io/mosn/pkg/filter/network/tunnel/ext"
)

func init() {
	ext.RegisterConnectionValidator("test", &CustomValidator{})
	ext.RegisterConnectionCredentialGetter("test", func(cluster string) string {
		return fmt.Sprintf("%s_%s", "prefix_", cluster)
	})
}

type CustomValidator struct {
}

func (c *CustomValidator) Validate(credential string, host string, cluster string) bool {
	return strings.HasPrefix(credential, "prefix_")
}
