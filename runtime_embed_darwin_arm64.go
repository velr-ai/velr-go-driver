//go:build !ios

package velr

import _ "embed"

//go:embed runtime/darwin-arm64/prebuilt/libvelrc.dylib.gz
var embeddedRuntime []byte
