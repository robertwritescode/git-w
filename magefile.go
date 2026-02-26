//go:build mage

package main

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = All

func All() {
	mg.Deps(Lint, Test, Build)
}

func Build() error {
	version := gitVersion()
	ldflags := fmt.Sprintf("-X main.version=%s", version)
	return sh.RunV("go", "build", "-ldflags="+ldflags, "-o", "bin/git-w", ".")
}

func Install() error {
	return sh.RunV("go", "install", ".")
}

func Test() error {
	return sh.RunV("go", "test", "-race", "-count=1", "./...")
}

func Cover() error {
	if err := sh.RunV("go", "test", "-race", "-count=1", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}
	return sh.RunV("go", "tool", "cover", "-html=coverage.out")
}

func Lint() error {
	if err := sh.RunV("golangci-lint", "fmt", "--diff", "./..."); err != nil {
		return err
	}
	return sh.RunV("golangci-lint", "run", "./...")
}

func LintFix() error {
	if err := sh.RunV("golangci-lint", "fmt", "./..."); err != nil {
		return err
	}
	return sh.RunV("golangci-lint", "run", "--fix", "./...")
}

func Fmt() error {
	return sh.RunV("golangci-lint", "fmt", "./...")
}

func gitVersion() string {
	out, err := sh.Output("git", "describe", "--tags", "--always", "--dirty")
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(out)
}
