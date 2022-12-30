// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shaper

import (
	"errors"
	"fmt"

	tc "github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/randomvariable/sqm/datastore"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

var ErrSNMPNotReady = errors.New("SNMP data not ready")

// Controller holds all local information.
type Controller struct {
	// ifbDevice defines whether this is the IFB device or not
	ifbDevice bool
	// data is the shared datastore
	data *datastore.Data
	// log is the logger
	log *zap.SugaredLogger
	// tcnl is the socket connection to rtnetlink
	tcnl *tc.Tc
}

const (
	averageOverhead      = uint32(68)
	defaultIngressHandle = uint32(0x8013)
	defaultEgressHandle  = uint32(0x8012)
	bitrateMultiplier    = 125
)

// NewShaperController returns an instantiated controller.
func NewShaperController(ifbDevice bool, data *datastore.Data, log *zap.SugaredLogger) (*Controller, error) {
	newLog := log.Named("Shaper controller").With("IsIfbDevice", ifbDevice)
	tcnl, err := tc.Open(&tc.Config{
		NetNS:  0,
		Logger: nil,
	})
	if err != nil {
		log.Errorw("unable to open netlink socket", "error", err)

		return nil, fmt.Errorf("unable to open netlink socket: %w", err)
	}

	ctrl := &Controller{
		ifbDevice: ifbDevice,
		data:      data,
		log:       newLog,
		tcnl:      tcnl,
	}

	return ctrl, nil
}

// device returns the appropriate root or IFB device.
func (c *Controller) device() (netlink.Link, error) {
	if c.ifbDevice {
		return c.data.IfbDevice() //nolint:wrapcheck
	}

	return c.data.RootDevice() //nolint:wrapcheck
}

// rate returns the appropriate ingress or egress rate.
func (c *Controller) rate() int64 {
	if c.ifbDevice {
		return c.data.IngressRate()
	}

	return c.data.EgressRate()
}

// baseHandle is an arbitrary base handle value depending on ingress or egress.
func (c *Controller) baseHandle() uint32 {
	if c.ifbDevice {
		return defaultIngressHandle
	}

	return defaultEgressHandle
}

// Reconcile defines the reconciliation loop.
func (c *Controller) Reconcile() error {
	device, err := c.device()
	if err != nil {
		return fmt.Errorf("could not get device: %w", err)
	}

	if c.rate() == 0 {
		c.log.Warn("SNMP data not ready")

		return ErrSNMPNotReady
	}

	baserate := uint64(c.rate() * bitrateMultiplier)
	kernelYes := uint32(1)
	diffservMode := uint32(0)
	overhead := averageOverhead
	cakeAtmPtm := uint32(2) //nolint:gomnd
	qdisc := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(device.Attrs().Index),
			Handle:  core.BuildHandle(c.baseHandle(), 0),
			Parent:  tc.HandleRoot,
			Info:    0,
		},
		Attribute: tc.Attribute{ //nolint:exhaustruct
			Kind: "cake",
			Cake: &tc.Cake{ //nolint:exhaustruct
				BaseRate:     &baserate,
				DiffServMode: &diffservMode,
				Nat:          &kernelYes,
				AckFilter:    &kernelYes,
				SplitGso:     &kernelYes,
				Overhead:     &overhead,
				Atm:          &cakeAtmPtm,
			},
		},
	}

	if err := c.tcnl.Qdisc().Replace(&qdisc); err != nil {
		c.log.Errorw("Could not assign qdisc to device", "error", err)

		return fmt.Errorf("could not assign qdisc to device: %w", err)
	}

	c.log.Info("Rebuilt cake qdisc")

	return nil
}

// ReconcileDelete defines what happens on shutdown.
func (c *Controller) ReconcileDelete() {
	// Not yet implemented
	c.log.Info("Shaper deletion not yet implemented")
}

//nolint:godox
// TODO @randomvariable: Set up these DSCP marks?
// iptables -A PREROUTING -i vtun+ -p tcp -j MARK --set-xmark 0x2/0xff
// iptables -A PREROUTING -i ppp0 -m dscp ! --dscp 0x00 -j DSCP --set-dscp 0x00
// iptables -A OUTPUT -p udp -m multiport --ports 123,53 -j DSCP --set-dscp 0x24
// iptables -A POSTROUTING -o ppp0 -m mark --mark 0x0/0xff -g QOS_MARK_ppp0
