// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

/*
links is a package for managing netlink links
*/
package links

import (
	"fmt"

	"github.com/randomvariable/sqm/datastore"

	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

// DeviceController defines the controller.
type DeviceController struct {
	// deviceName is the ip link name
	deviceName string
	// data is the shared datastore
	data *datastore.Data
	// log is a logger
	log *zap.SugaredLogger
	// create defines whether or not this controller is managing an IFB device or the root device
	create bool
}

// NewDeviceController creates an instantiated controller.
func NewDeviceController(deviceName string,
	data *datastore.Data, create bool, log *zap.SugaredLogger,
) DeviceController {
	if create {
		deviceName = "ifb4" + deviceName
	}

	return DeviceController{
		deviceName: deviceName,
		data:       data,
		log:        log.Named("Device Controller").With("DeviceName", deviceName),
		create:     create,
	}
}

// ensureUp ensures the link is up or at least Unknown.
func (d DeviceController) ensureUp(dev netlink.Link) error {
	if dev.Attrs().OperState != netlink.OperUp && dev.Attrs().OperState != netlink.OperUnknown {
		d.log.Warnw("Device is not up, bringing up", "OperState", dev.Attrs().OperState.String())

		if err := netlink.LinkSetUp(dev); err != nil {
			return fmt.Errorf("error setting operational state: %w", err)
		}
	}

	return nil
}

// Reconcile defines the reconciliation loop.
func (d DeviceController) Reconcile() error {
	dev, err := netlink.LinkByName(d.deviceName)
	if err != nil {
		if d.create {
			return d.tryCreate()
		}

		return fmt.Errorf("error retrieving device link: %w", err)
	}

	updated := false

	if d.create {
		if err := d.ensureUp(dev); err != nil {
			return err
		}

		updated = d.data.SetIfbDevice(dev)
	} else {
		updated = d.data.SetRootDevice(dev)
	}

	if updated {
		d.log.Infow("Updated device", "DeviceIndex", dev.Attrs().Index, "DeviceMTU", dev.Attrs().MTU)
	}

	return nil
}

// tryCreate attempts to create a new IFB link.
func (d DeviceController) tryCreate() error {
	ifbLink := netlink.GenericLink{
		LinkAttrs: netlink.NewLinkAttrs(),
		LinkType:  "ifb",
	}
	ifbLink.LinkAttrs.Name = d.deviceName

	rootDevice, err := d.data.RootDevice()
	if err != nil {
		return fmt.Errorf("cannot create IFB device without root device data: %w", err)
	}

	ifbLink.LinkAttrs.MTU = rootDevice.Attrs().MTU
	if err := netlink.LinkAdd(&ifbLink); err != nil {
		return fmt.Errorf("cannot add IFB device: %w", err)
	}

	return nil
}

// ReconcileDelete defines what happens on shutdown.
func (d DeviceController) ReconcileDelete() {
	if !d.create {
		return
	}

	ifbDevice, err := d.data.IfbDevice()
	if err != nil {
		return
	}

	if err := netlink.LinkDel(ifbDevice); err != nil {
		d.log.Errorw("Cannot delete IFB device", "error", err)
	}

	d.log.Info("Torn down device")
}
