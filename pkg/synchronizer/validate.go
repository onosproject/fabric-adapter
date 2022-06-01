// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements declarations and utilities for the synchronizer.
package synchronizer

import (
	"fmt"
)

// Validation functions, return an error if the given struct is missing data that
// prevents synchronization.

func validateRoute(route *Route) error {
	if (route.Prefix == nil) || (*route.Prefix == "") {
		return fmt.Errorf("Route %s has no Prefix", *route.RouteId)
	}
	if (route.Address == nil) || (*route.Address == "") {
		return fmt.Errorf("Route %s has no Address", *route.RouteId)
	}
	return nil
}
