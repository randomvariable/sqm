// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:gochecknoglobals
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/randomvariable/sqm/links"
	"github.com/randomvariable/sqm/manager"
	"github.com/randomvariable/sqm/redirector"
	"github.com/randomvariable/sqm/shaper"
	"github.com/randomvariable/sqm/snmp"
	"github.com/spf13/cobra"
)

var (
	rootDevice string
	ingressOID string
	egressOID  string
	snmpHost   string
)

const (
	shortTickerSeconds = 5
	longTickerSeconds  = 60
)

var rootCmd = generateNewRoot()

// RootCmd is the Cobra root command.
func generateNewRoot() *cobra.Command {
	newCmd := &cobra.Command{ //nolint:exhaustruct
		Use:   "sqm",
		Short: "sqm",
		Long: LongDesc(`
			sqm sets up the Cake scheduler bi-directionally, and can introspect modems via SNMP to update bandwidth targets.
		`),
		Example: Examples(`
			sqm --interface ppp0
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := manager.NewManager()
			if err != nil {
				return fmt.Errorf("cannot create manager: %w", err)
			}
			snmpController := snmp.NewSNMPController(ingressOID, egressOID, snmpHost, mgr.Data, mgr.Log)
			mgr.AddController("SNMP", snmpController, time.Second*shortTickerSeconds)
			rootDeviceController := links.NewDeviceController(rootDevice, mgr.Data, false, mgr.Log)
			mgr.AddController("Root Device", rootDeviceController, time.Second*shortTickerSeconds)
			ifbDeviceController := links.NewDeviceController(rootDevice, mgr.Data, true, mgr.Log)
			mgr.AddController("IFB Device", ifbDeviceController, time.Second*shortTickerSeconds)
			rootShaperController, err := shaper.NewShaperController(false, mgr.Data, mgr.Log)
			if err != nil {
				os.Exit(1)
			}
			mgr.AddController("Root Device Shaper", rootShaperController, time.Second*longTickerSeconds)
			ifbShaperController, err := shaper.NewShaperController(true, mgr.Data, mgr.Log)
			if err != nil {
				os.Exit(1)
			}
			mgr.AddController("IFB Device Shaper", ifbShaperController, time.Second*longTickerSeconds)
			redirectController := redirector.NewRedirectorController(mgr.Data, mgr.Log)
			mgr.AddController("Redirector", redirectController, time.Second*longTickerSeconds)

			return mgr.Start() //nolint:wrapcheck
		},
		Args: cobra.NoArgs,
	}

	newCmd.PersistentFlags().StringVarP(&rootDevice, "interface", "d", "ppp0", "Device to configure")
	newCmd.PersistentFlags().StringVarP(&ingressOID, "--ingress-oid", "i",
		snmp.ZyxelSNMPIngressOID, "SNMP OID for ingress")
	newCmd.PersistentFlags().StringVarP(&egressOID, "--egress-oid", "e", snmp.ZyxelSNMPEgressOID, "SNMP OID for egress")
	newCmd.PersistentFlags().StringVarP(&snmpHost, "--snmp-host", "l", "192.168.2.1", "SNMP Host")

	return newCmd
}

// Execute starts the process.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
