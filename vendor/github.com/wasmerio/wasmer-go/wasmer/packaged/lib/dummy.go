// See https://github.com/golang/go/issues/26366.
package lib

import (
	_ "github.com/wasmerio/wasmer-go/wasmer/packaged/lib/darwin-amd64"
	_ "github.com/wasmerio/wasmer-go/wasmer/packaged/lib/linux-aarch64"
	_ "github.com/wasmerio/wasmer-go/wasmer/packaged/lib/linux-amd64"
)
