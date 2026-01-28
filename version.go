package main

import (
	"fmt"
	"runtime"

	"github.com/sapcc/mosquitto-exporter/internal/version"
)

func versionString() string {
	return fmt.Sprintf("%s (%s), %s", version.Version, version.Commit, runtime.Version())
}
