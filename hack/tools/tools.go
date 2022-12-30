// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: typecheck
// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "github.com/aquasecurity/trivy/cmd/trivy"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "mvdan.cc/gofumpt"
)
