//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = All

func All() {
	mg.Deps(Vet, Test, Build)
}

func Build() error {
	version := gitVersion()
	ldflags := fmt.Sprintf("-X main.version=%s", version)
	return sh.RunV("go", "build", "-ldflags="+ldflags, "-o", "bin/git-workspace", ".")
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

func Vet() error {
	return sh.RunV("go", "vet", "./...")
}

func gitVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(out))
}
