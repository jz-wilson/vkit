package main

import "github.com/jz-wilson/vkit/cmd"

// version is overridden at release time via goreleaser:
// -ldflags "-X main.version={{.Version}}".
var version = "dev"

func main() {
	cmd.Execute(version)
}
