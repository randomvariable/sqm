// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package datastore

import (
	"errors"
	"sync"

	"github.com/vishvananda/netlink"
)

var ErrDeviceNotYetReady = errors.New("device not yet ready")

type Data struct {
	mu          sync.Mutex
	ingressRate int64
	egressRate  int64
	rootDevice  netlink.Link
	ifbDevice   netlink.Link
}

func NewDataStore() *Data {
	return &Data{
		mu:          sync.Mutex{},
		ingressRate: 0,
		egressRate:  0,
		rootDevice:  nil,
		ifbDevice:   nil,
	}
}

func (d *Data) IngressRate() int64 {
	return d.ingressRate
}

func (d *Data) EgressRate() int64 {
	return d.egressRate
}

func (d *Data) RootDevice() (netlink.Link, error) {
	if d.rootDevice == nil {
		return nil, ErrDeviceNotYetReady
	}

	return d.rootDevice, nil
}

func (d *Data) IfbDevice() (netlink.Link, error) {
	if d.ifbDevice == nil {
		return nil, ErrDeviceNotYetReady
	}

	return d.ifbDevice, nil
}

func (d *Data) SetIngressRate(newVal int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.ingressRate != newVal {
		d.ingressRate = newVal

		return true
	}

	return false
}

func (d *Data) SetEgressRate(newVal int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.egressRate != newVal {
		d.egressRate = newVal

		return true
	}

	return false
}

func isSameDevice(old netlink.Link, newLink netlink.Link) bool {
	if old == nil {
		return false
	}

	if old.Attrs().Index != newLink.Attrs().Index {
		return false
	}

	if old.Attrs().MTU != newLink.Attrs().MTU {
		return false
	}

	return true
}

func (d *Data) SetRootDevice(newLink netlink.Link) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !isSameDevice(d.rootDevice, newLink) {
		d.rootDevice = newLink

		return true
	}

	return false
}

func (d *Data) SetIfbDevice(newLink netlink.Link) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !isSameDevice(d.ifbDevice, newLink) {
		d.ifbDevice = newLink

		return true
	}

	return false
}
