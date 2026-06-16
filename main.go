package main

import (
	"fmt"

	"github.com/jz-wilson/vkit/cmd"
)

// version is overridden at release time via goreleaser:
// -ldflags "-X main.version={{.Version}}".
var version = "dev"
var commit = "none"
var buildDate = "unknown"

func main() {
	v := version
	if commit != "none" {
		v = fmt.Sprintf("%s (%s %s)", version, commit, buildDate)
	}
	cmd.Execute(v)
}
