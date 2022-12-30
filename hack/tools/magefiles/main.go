// nolint: wrapcheck
package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
	c "github.com/randomvariable/sqm/magefiles/common"
)

func Module() error {
	return c.Module()
}

func toolUpdate(path string) (bool, error) {
	changes, err := c.HaveModulesChanged()
	if err != nil {
		return false, err
	}

	globs, err := c.GoGlobsWithModules()
	if err != nil {
		return false, err
	}

	toolUpdate, err := target.Path(path, globs...)
	if err != nil {
		return false, err
	}

	return changes || toolUpdate, nil
}

func buildGoTool(binpath, gopath string) error {
	mg.Deps(Module)

	needsUpdate, err := toolUpdate(binpath)
	if err != nil {
		return err
	}

	if !needsUpdate {
		return nil
	}

	return sh.Run("go", "build", "-o", binpath, gopath)
}

func GolangCILint() error {
	return buildGoTool("bin/golangci-lint", "github.com/golangci/golangci-lint/cmd/golangci-lint")
}

func Trivy() error {
	return buildGoTool("bin/trivy", "github.com/aquasecurity/trivy/cmd/trivy")
}

func GoFumpt() error {
	return buildGoTool("bin/gofumpt", "mvdan.cc/gofumpt")
}
