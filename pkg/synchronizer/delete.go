// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

/*
 * Deletes are always handled synchronously. The synchronizer stops and wait for a delete to be
 * completed before it continues. This is to handle the case where we fail while performing the
 * delete. It'll get marked as a FAILED transaction in onos-config.
 *
 * This is in contrasts to configuration updates, which are generally handled asynchronously.
 */

import (
	"github.com/onosproject/sdcore-adapter/pkg/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// HandleDelete synchronously performs a delete
func (s *Synchronizer) HandleDelete(config *gnmi.ConfigForest, path *pb.Path) error {
	// TODO: implement
	return nil
}
