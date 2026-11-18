//go:build !ios

package velr

import _ "embed"

//go:embed runtime/darwin-x64/prebuilt/libvelrc.dylib.gz
var embeddedRuntime []byte
