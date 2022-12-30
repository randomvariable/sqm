// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package redirector

import (
	"fmt"
	"syscall"

	"github.com/florianl/go-tc/core"
	"github.com/randomvariable/sqm/datastore"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

// Controller contains all local information required for reconciliation.
type Controller struct {
	// data is the shared datastore
	data *datastore.Data
	// log is the logger
	log *zap.SugaredLogger
}

const (
	DefaultPriority = 10
	rootHandleBase  = uint16(0xffff)
	rootHandleSub   = uint16(0)
)

// NewRedirectorController returns an instantiated controller.
func NewRedirectorController(data *datastore.Data, log *zap.SugaredLogger) *Controller {
	return &Controller{
		data: data,
		log:  log.Named("Redirector Controller"),
	}
}

// Reconcile defines the reconciliation loop.
func (c *Controller) Reconcile() error {
	rootDev, err := c.data.RootDevice()
	if err != nil {
		return fmt.Errorf("error getting root device: %w", err)
	}

	c.log = c.log.With("RootDevice", rootDev.Attrs().Name)
	redirectDevice, err := c.data.IfbDevice()
	if err != nil {
		return fmt.Errorf("error getting root device: %w", err)
	}

	c.log = c.log.With("IFBDevice", redirectDevice.Attrs().Name)

	qdisc, err := c.reconcileRootQdisc()
	if err != nil {
		return err
	}

	return c.reconcileRedirect(qdisc.Attrs().Handle)
}

// reconcileRedirect sets up the tc filter using the mirred action to redirect ingress traffic from
// the root device to the IFB device so it can be shaped.
func (c *Controller) reconcileRedirect(qdiscHandle uint32) error {
	rootDevice, err := c.data.RootDevice()
	if err != nil {
		return fmt.Errorf("error getting root device: %w", err)
	}

	ifbDevice, err := c.data.IfbDevice()
	if err != nil {
		return fmt.Errorf("error getting ifb device: %w", err)
	}

	filters, err := netlink.FilterList(rootDevice, netlink.HANDLE_INGRESS)
	if err != nil {
		return fmt.Errorf("error getting filters for root device: %w", err)
	}

	for _, filter := range filters {
		if filter.Type() == "u32" {
			return nil
		}
	}

	if err := netlink.FilterAdd(&netlink.U32{
		ClassId: core.BuildHandle(1, 1),
		Link:    0,
		FilterAttrs: netlink.FilterAttrs{
			Parent:    qdiscHandle,
			Protocol:  syscall.ETH_P_ALL,
			Priority:  DefaultPriority,
			LinkIndex: rootDevice.Attrs().Index,
			Handle:    0,
		},
		Sel:        nil,
		Divisor:    0,
		Hash:       0,
		RedirIndex: 0,
		Actions: []netlink.Action{
			&netlink.MirredAction{
				MirredAction: netlink.TCA_EGRESS_REDIR,
				Ifindex:      ifbDevice.Attrs().Index,
				ActionAttrs: netlink.ActionAttrs{
					Action:  netlink.TC_ACT_STOLEN,
					Index:   0,
					Capab:   0,
					Refcnt:  0,
					Bindcnt: 0,
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("error adding redirect filter: %w", err)
	}

	c.log.Info("Added redirect filter")

	return nil
}

// findRootQdisc attempts to find the root qdisc to which the redirection will be
// set up upon.
func (c *Controller) findRootQdisc() (netlink.Qdisc, bool) {
	var rootQDisc netlink.Qdisc

	rootDevice, err := c.data.RootDevice()
	if err != nil {
		return rootQDisc, false
	}

	qdiscs, err := netlink.QdiscList(rootDevice)
	if err != nil {
		c.log.Errorw("Error retreiving qdisc list for root device", "error", err)

		return rootQDisc, false
	}

	for _, qdisc := range qdiscs {
		if qdisc.Attrs().Parent == netlink.HANDLE_INGRESS {
			rootQDisc = qdisc
		}
	}

	if rootQDisc == nil {
		return rootQDisc, false
	}

	return rootQDisc, true
}

// reconcileRootQdisc ensures there's a root qdisc handle for redirection.
func (c *Controller) reconcileRootQdisc() (netlink.Qdisc, error) {
	qdisc, ok := c.findRootQdisc()
	if !ok {
		if err := c.createRootHandle(); err != nil {
			return qdisc, err
		}

		return c.reconcileRootQdisc()
	}

	return qdisc, nil
}

// createRootHandle creates a root qdisc handle.
func (c *Controller) createRootHandle() error {
	rootDevice, err := c.data.RootDevice()
	if err != nil {
		return fmt.Errorf("error getting root device: %w", err)
	}

	if err := netlink.QdiscReplace(&netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: rootDevice.Attrs().Index,
			Handle:    netlink.MakeHandle(rootHandleBase, rootHandleSub),
			Parent:    netlink.HANDLE_INGRESS,
			Refcnt:    uint32(1),
		},
	}); err != nil {
		return fmt.Errorf("error creating root handle: %w", err)
	}

	c.log.Infow("Created root handle")

	return nil
}

// ReconcileDelete defines what happens on shutdown.
func (c *Controller) ReconcileDelete() {
	qdisc, ok := c.findRootQdisc()
	if !ok {
		c.log.Info("Couldn't find root handle. Skipping")

		return
	}

	if err := netlink.QdiscDel(qdisc); err != nil {
		c.log.Errorf("Error deleting root qdisc for redirection: %v", err)
	}

	c.log.Info("Torn down redirection")
}
