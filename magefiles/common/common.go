// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package common defines common functions for the magefiles.
package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
	"github.com/magefile/mage/target"
)

const (
	goModFile = "go.mod"
	goSumFile = "go.sum"
)

func currentDir() (string, error) {
	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current directory: %w", err)
	}

	return path, nil
}

func GoGlobs() ([]string, error) {
	globs, err := filepath.Glob("**/*.go")
	if err != nil {
		return nil, fmt.Errorf("could not gather **/*.go globs: %w", err)
	}

	return globs, nil
}

func GoGlobsWithModules() ([]string, error) {
	globs, err := GoGlobs()
	if err != nil {
		return nil, err
	}

	return append(globs, goModFile, goSumFile), nil
}

// nolint: forbidigo,nolintlint
func HaveModulesChanged() (bool, error) {
	goModuleNeedsUpdating, err := target.Glob(goModFile, "**/*.go")
	if err != nil {
		return false, err //nolint:wrapcheck
	}

	if goModuleNeedsUpdating {
		fmt.Println("Files changed against go.mod")
	}

	goSumNeedsUpdating, err := target.Glob(goSumFile, "**/*.go")
	if err != nil {
		return false, err //nolint:wrapcheck
	}

	if goSumNeedsUpdating {
		fmt.Println("Files changed against go.sum")
	}

	goModuleChanged, err := target.Glob(goSumFile, goModFile)
	if err != nil {
		return false, err //nolint:wrapcheck
	}

	if goModuleChanged {
		fmt.Println("go.mod changed")
	}

	changes := goModuleNeedsUpdating || goSumNeedsUpdating || goModuleChanged

	return changes, nil
}

// nolint: forbidigo,nolintlint
func Module() error {
	changes, err := HaveModulesChanged()
	if err != nil {
		return err
	}

	if !changes {
		return nil
	}

	path, err := currentDir()
	if err != nil {
		return err
	}

	fmt.Println("Tidying modules", "path", path)

	err = sh.Run("go", "mod", "tidy")
	if err != nil {
		return fmt.Errorf("unable to tidy go modules: %w", err)
	}

	return sh.Run("touch", goModFile, goSumFile) //nolint:wrapcheck
}
