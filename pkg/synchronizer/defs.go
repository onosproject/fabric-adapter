// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements the synchronizer.
package synchronizer

import (
	"github.com/atomix/atomix-go-client/pkg/atomix/counter"
	"time"

	"github.com/onosproject/sdcore-adapter/pkg/gnmi"
	"github.com/onosproject/sdcore-adapter/pkg/metrics"
)

const (
	// DefaultImsiFormat is the default format for an IMSI string
	DefaultImsiFormat = "CCCNNNEEESSSSSS"

	// DefaultPostTimeout is the default timeout for post operations
	DefaultPostTimeout = time.Second * 10

	// DefaultPartialUpdateEnable is the default partial update setting
	DefaultPartialUpdateEnable = true

	// SidCounter is the name used for atomix counter for generating unique SIDs
	SidCounter = "fabric-adapter-sid-counter"

	// SidMap is the name used for atomix counter for generating unique SIDs
	SidMap = "fabric-adapter-sid-map"
)

// Synchronizer is a Version 3 synchronizer.
type Synchronizer struct {
	postEnable          bool
	postTimeout         time.Duration
	pusher              PusherInterface
	updateChannel       chan *ConfigUpdate
	retryInterval       time.Duration
	partialUpdateEnable bool
	caPath              string
	keyPath             string
	certPath            string
	topoEndpoint        string

	// Busy indicator, primarily used for unit testing. The channel length in and of itself
	// is not sufficient, as it does not include the potential update that is currently syncing.
	// >0 if the synchronizer has operations pending and/or in-progress
	busy int32

	// used for ease of mocking
	synchronizeDeviceFunc func(config *gnmi.ConfigForest) (int, error)

	// cache of previously synchronized updates
	cache map[string]interface{}

	// Prometheus fetchers for each endpoint
	prometheus map[string]*metrics.Fetcher

	kafkaMsgChannel   chan string
	kafkaErrorChannel chan error

	nextSID counter.Counter
	sidMap  map[string]uint32
}

// ConfigUpdate holds the configuration for a particular synchronization request
type ConfigUpdate struct {
	config       *gnmi.ConfigForest
	callbackType gnmi.ConfigCallbackType
	target       string
}

// SynchronizerOption is for options passed when creating a new synchronizer
type SynchronizerOption func(c *Synchronizer) // nolint

// FabricScope is used within the synchronizer to convey the scope we're working at within the
// tree. Contexts were considered for this implementation, but rejected due to the lack of
// static checking.
type FabricScope struct {
	FabricId        *string      // nolint - use EnterpriseId to match the ygot naming convention
	Fabric          *RootDevice  // Each fabric is a configuration tree
	Switch          *Switch      // The switch we're currently working on
	SwitchModel     *SwitchModel // The switch model we're currently working on
	OnosEndpoint    *string      // Endpoint of Onos to post to
	OnosUsername    *string      // Username for authenticating to ONOS
	OnosPassword    *string      // Password for authenticating to ONOS
	StratumEndpoint *string      // Endpoint of Fabric to post to
	NetConfig       *OnosNetConfig
}
