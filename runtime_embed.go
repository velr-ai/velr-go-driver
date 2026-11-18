package velr

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func resolveBundledRuntime() (string, error) {
	platform, ok := currentRuntimePlatform()
	if !ok {
		return "", nil
	}

	archive, err := gzip.NewReader(bytes.NewReader(embeddedRuntime))
	if err != nil {
		return "", fmt.Errorf("open bundled Velr runtime: %w", err)
	}
	defer archive.Close()

	library, err := io.ReadAll(archive)
	if err != nil {
		return "", fmt.Errorf("decompress bundled Velr runtime: %w", err)
	}
	return materializeRuntime(library, platform.names[0])
}
