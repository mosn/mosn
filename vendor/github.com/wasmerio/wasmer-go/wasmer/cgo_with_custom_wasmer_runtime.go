// +build custom_wasmer_runtime

package wasmer

// // With the `customlib` tag, the user is expected to provide the
// // `CGO_LDFLAGS` and `CGO_CFLAGS` that point to appropriate Wasmer build
// // directories, e.g.:
// //
// // ```sh
// // export CGO_CFLAGS="-I/wasmer/lib/c-api/"
// //
// // export CGO_LDFLAGS="-Wl,-rpath,/wasmer/target/x86_64-unknown-linux-musl/release/ -L/wasmer/target/x86_64-unknown-linux-musl/release/ -pthread -lwasmer_c_api -lm -ldl -static"
// //
// // export CC=/usr/bin/musl-gcc
// //
// // go build -tags custom_wasmer_runtime
// // ```
//
// #include <wasmer_wasm.h>
import "C"
