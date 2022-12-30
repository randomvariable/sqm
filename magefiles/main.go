// nolint: forbidigo, wrapcheck
package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
	c "github.com/randomvariable/sqm/magefiles/common"
)

func Module() error {
	return c.Module()
}

func Modules() error {
	mg.Deps(Module)

	if err := sh.Run("sh", "-c", "cd hack/tools && mage module"); err != nil {
		return fmt.Errorf("failed to tidy modules in hack/tools directory: %w", err)
	}

	globs, err := c.GoGlobsWithModules()
	if err != nil {
		return fmt.Errorf("failed to get go globs: %w", err)
	}

	workspaceNeedsUpdating, err := target.Path("go.work.sum", globs...)
	if err != nil {
		return fmt.Errorf("failed to check if workspace needs updating: %w", err)
	}

	if workspaceNeedsUpdating {
		fmt.Println("Syncing go workspace")

		err := sh.Run("go", "work", "sync")
		if err != nil {
			return fmt.Errorf("failed to sync go workspace: %w", err)
		}

		return sh.Run("touch", "go.work.sum")
	}

	return nil
}

func Install() error {
	mg.Deps(Module)

	return sh.Run("go", "install", "./cmd/sqm")
}

func SecurityScan() error {
	mg.Deps(Module)

	err := sh.Run("sh", "-c", "cd hack/tools && mage trivy")
	if err != nil {
		return fmt.Errorf("failed to build trivy: %w", err)
	}

	fmt.Println("Running trivy...")

	_, err = sh.Exec(nil, os.Stdout, os.Stderr, "hack/tools/bin/trivy", "filesystem", ".")

	if err != nil {
		return fmt.Errorf("failed to run trivy: %w", err)
	}

	return nil
}

func Fmt() error {
	mg.Deps(Module)

	err := sh.Run("sh", "-c", "cd hack/tools && mage gofumpt")
	if err != nil {
		return fmt.Errorf("failed to build gofumpt: %w", err)
	}

	fmt.Println("Running gofumpt...")

	return sh.Run("hack/tools/bin/gofumpt", "-w", ".")
}

func Lint() error {
	mg.Deps(Module)

	err := sh.Run("sh", "-c", "cd hack/tools && mage golangCILint")
	if err != nil {
		return fmt.Errorf("failed to build golangci-lint: %w", err)
	}

	fmt.Println("Running golangci-lint...")

	_, err = sh.Exec(nil, os.Stdout, os.Stderr, "hack/tools/bin/golangci-lint", "run")

	if err != nil {
		return fmt.Errorf("failed to run golangci-lint: %w", err)
	}

	return nil
}
