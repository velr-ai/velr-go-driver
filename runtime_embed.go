package velr

import "embed"

// embeddedRuntime contains the platform runtime payloads shipped with the Go driver.
//
//go:embed runtime/*/prebuilt/* runtime/*/LICENSE.runtime
var embeddedRuntime embed.FS
