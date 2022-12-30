// Copyright 2022 Naadir Jeewa. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/randomvariable/sqm/datastore"
	"go.uber.org/zap"
)

// Manager defines the overall runtime manager.
type Manager struct {
	Data        *datastore.Data
	Log         *zap.SugaredLogger
	tickers     []*time.Ticker
	wg          *sync.WaitGroup
	done        chan bool
	controllers []controllerInfo
}

// NewManager returns an instantiated manager.
func NewManager() (*Manager, error) {
	logConfig := zap.NewProductionConfig()
	logConfig.DisableStacktrace = true
	logger, err := logConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to set up logger: %w", err)
	}

	mgr := &Manager{
		Data:        datastore.NewDataStore(),
		wg:          &sync.WaitGroup{},
		done:        make(chan bool),
		Log:         logger.Sugar(),
		tickers:     []*time.Ticker{},
		controllers: []controllerInfo{},
	}

	return mgr, nil
}

// newTicker sets up the reconciliation loops for each controller.
func (m *Manager) newTicker(name string,
	done chan bool,
	waitgroup *sync.WaitGroup,
	duration time.Duration,
	reconcileFunc func() error,
	deleteFunc func(),
) *time.Ticker {
	waitgroup.Add(1)

	ticker := time.NewTicker(duration)

	go func() {
		for {
			select {
			case <-done:
				deleteFunc()
				waitgroup.Done()

				return
			case <-ticker.C:
				m.checkReconciliationWithError(name, reconcileFunc)
			}
		}
	}()

	return ticker
}

// nolint: godox,nolintlint
// setupCloseHandler traps Ctrl+C then runs each controller's delete reconciliation.
func (m *Manager) setupCloseHandler(done chan bool) {
	chnl := make(chan os.Signal)

	/* TODO @randomvariable: sigchanyzer: misuse of unbuffered os.Signal channel as
	argument to signal.Notify (govet)go-golangci-lint */

	signal.Notify(chnl, os.Interrupt, syscall.SIGTERM) //nolint:govet

	go func() {
		<-chnl
		m.Log.Info("Shutting down")

		for _, ticker := range m.tickers {
			ticker.Stop()
		}

		close(done)
		m.wg.Wait()
		os.Exit(0)
	}()
}

// controller defines the minimal interface every controller should have.
type controller interface {
	Reconcile() error
	ReconcileDelete()
}

// controllerInfo stores information about the controllers prior to being started.
type controllerInfo struct {
	name           string
	tickerDuration time.Duration
	controller     controller
}

// AddController adds a controller for start up.
func (m *Manager) AddController(name string, c controller, t time.Duration) {
	m.controllers = append(m.controllers, controllerInfo{name: name, controller: c, tickerDuration: t})
}

func (m *Manager) checkReconciliationWithError(name string, f func() error) {
	if err := f(); err != nil {
		log := m.Log.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar().With("controller", name)
		log.Error(err)
	}
}

// startController starts reconciliation of a controller.
func (m *Manager) startController(name string, ctrller controller, t time.Duration) {
	m.tickers = append(m.tickers, m.newTicker(
		name,
		m.done,
		m.wg,
		t,
		ctrller.Reconcile,
		ctrller.ReconcileDelete,
	))

	m.checkReconciliationWithError(name, ctrller.Reconcile)
	m.Log.Info("Started " + name + "controller")
}

// Start starts all controllers.
func (m *Manager) Start() error {
	m.Log.Info("Starting controllers...")

	for _, c := range m.controllers {
		m.startController(c.name, c.controller, c.tickerDuration)
	}

	m.setupCloseHandler(m.done)
	runtime.Goexit()

	return nil
}
