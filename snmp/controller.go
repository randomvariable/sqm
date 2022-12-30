// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package snmp

import (
	"fmt"

	"github.com/gosnmp/gosnmp"
	"github.com/randomvariable/sqm/datastore"
	"go.uber.org/zap"
)

// snmpController defines the local information for this controller.
type Controller struct {
	// ingressOID is the SNMP OID for reading the ingress rate in kbps
	ingressOID string
	// ingressOID is the SNMP OID for reading the egress rate in kbps
	egressOID string
	// data is the shared datastore
	data *datastore.Data
	// host is the SNMP host to read from
	host string
	// log is the logger
	log *zap.SugaredLogger
}

// NewSNMPController returns an instantiated SNMP controller.
func NewSNMPController(ingressOID, egressOID, host string, data *datastore.Data, log *zap.SugaredLogger) Controller {
	ctrl := Controller{
		ingressOID: ingressOID,
		egressOID:  egressOID,
		host:       host,
		data:       data,
		log:        log.Named("SNMP Reader").With("Host", host, "Egress OID", egressOID, "Ingress OID", ingressOID),
	}

	return ctrl
}

// Reconcile defines the reconciliation loop.
func (s Controller) Reconcile() error {
	gosnmp.Default.Target = s.host
	err := gosnmp.Default.Connect()
	if err != nil {
		s.log.Errorw("Cannot connect", "error", err)

		return fmt.Errorf("cannot connect to SNMP host: %w", err)
	}

	oids := []string{s.ingressOID, s.egressOID}

	result, err := gosnmp.Default.Get(oids)
	if err != nil {
		s.log.Errorw("Cannot read SNMP", "error", err)

		return fmt.Errorf("cannot read SNMP: %w", err)
	}

	ingressRate := gosnmp.ToBigInt(result.Variables[0].Value).Int64()
	egressRate := gosnmp.ToBigInt(result.Variables[1].Value).Int64()
	ingressUpdated := s.data.SetIngressRate(ingressRate)
	egressUpdated := s.data.SetEgressRate(egressRate)

	if ingressUpdated || egressUpdated {
		s.log.Infow(
			"Read rates updated",
			"ingressUpdated",
			ingressUpdated,
			"ingress",
			ingressRate,
			"egressUpdated",
			egressUpdated,
			"egress",
			egressRate)
	}

	return nil
}

// ReconcileDelete defines what happens on shutdown.
func (s Controller) ReconcileDelete() {
	// Nothing to do
	s.log.Info("SNMP shut down")
}
