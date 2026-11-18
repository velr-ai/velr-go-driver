package velr

import _ "embed"

//go:embed runtime/win32-x64-msvc/prebuilt/velrc.dll.gz
var embeddedRuntime []byte
