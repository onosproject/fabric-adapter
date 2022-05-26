// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

import (
	"github.com/onosproject/sdcore-adapter/pkg/gnmi"
	"time"
)

// SynchronizeDevice synchronizes a device. Two sets of error state are returned:
//   1) pushFailures -- a count of pushes that failed to the core. Synchronizer should retry again later.
//   2) error -- a fatal error that occurred during synchronization.
func (s *Synchronizer) SynchronizeDevice(allConfig *gnmi.ConfigForest) (int, error) {
	pushFailures := 0
	for fabricID, fabricConfig := range allConfig.Configs {
		device := fabricConfig.(*RootDevice)

		tStart := time.Now()
		KpiSynchronizationTotal.WithLabelValues(fabricID).Inc()

		scope := &FabricScope{
			FabricId: &fabricID,
			Fabric:   device}

		// TODO: Everything
		_ = scope

		KpiSynchronizationDuration.WithLabelValues(fabricID).Observe(time.Since(tStart).Seconds())
	}

	return pushFailures, nil
}
