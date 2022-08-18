// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer is basic declarations and utilities for the synchronizer
package synchronizer

import (
	"fmt"
	"github.com/onosproject/sdcore-adapter/pkg/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// SynchronizerInterface defines the interface that all synchronizers should have.
type SynchronizerInterface interface { //nolint
	Synchronize(config *gnmi.ConfigForest, callbackType gnmi.ConfigCallbackType, target string, path *pb.Path) error
	GetModels() *gnmi.Model
	Start()
}

// PusherInterface is an interface to a pusher, which pushes json to underlying services.
//go:generate mockgen -destination=../test/mocks/mock_pusher.go -package=mocks github.com/onosproject/sdcore-adapter/pkg/synchronizer PusherInterface
type PusherInterface interface {
	PushUpdate() error
	PushDelete() error
}

// PushError is an error class that is returned for failed POSTs and DELETEs. It
// makes it easier to detect a nonfatal error, such as a 404.
type PushError struct {
	Endpoint   string
	StatusCode int
	Status     string
	Operation  string
}

func (e *PushError) Error() string {
	return fmt.Sprintf("Push Error op=%s endpoint=%s code=%d status=%s", e.Operation, e.Endpoint, e.StatusCode, e.Status)
}
