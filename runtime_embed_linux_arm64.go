//go:build !android

package velr

import _ "embed"

//go:embed runtime/linux-arm64-gnu/prebuilt/libvelrc.so.gz
var embeddedRuntime []byte
