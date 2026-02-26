package main

import "github.com/robertwritescode/git-w/pkg/cmd"

// version is set at build time via ldflags (e.g. -X main.version=v0.1.0).
var version string

func main() {
	cmd.Execute(version)
}
