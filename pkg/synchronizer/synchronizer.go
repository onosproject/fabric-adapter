// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for Aether models
package synchronizer

import (
	"context"
	"github.com/atomix/atomix-go-client/pkg/atomix"
	models "github.com/onosproject/config-models/models/sdn-fabric-0.1.x/api"
	"github.com/onosproject/fabric-adapter/pkg/store"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/sdcore-adapter/pkg/gnmi"
	"github.com/onosproject/sdcore-adapter/pkg/metrics"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	"os"
	"reflect"
	"time"
)

var log = logging.GetLogger("synchronizer")

// Synchronize synchronizes the state to the underlying service.
func (s *Synchronizer) Synchronize(config *gnmi.ConfigForest, callbackType gnmi.ConfigCallbackType, target string, path *pb.Path) error {
	var err error
	if callbackType == gnmi.Deleted {
		return s.HandleDelete(config, path)
	}

	if callbackType == gnmi.Forced {
		s.CacheInvalidate() // invalidate the post cache if this resync was forced by Diagnostic API
	}

	err = s.enqueue(config, callbackType, target)
	return err
}

// SynchronizeAndRetry automatically retries if synchronization fails
func (s *Synchronizer) SynchronizeAndRetry(ctx context.Context, update *ConfigUpdate) {
	for {
		// If something new has come along, then don't bother with the one we're working on
		if s.newUpdatesPending() {
			log.Infof("Current synchronizer update has been obsoleted")
			return
		}

		pushErrors, err := s.synchronizeDeviceFunc(ctx, update.config)
		if err != nil {
			log.Errorf("Synchronization error: %v", err)
			return
		}

		if pushErrors == 0 {
			log.Infof("Synchronization success")
			return
		}

		log.Infof("Synchronization encountered %d push errors, scheduling retry", pushErrors)

		// We failed to push something to the core. Sleep before trying again.
		// Implements a fixed interval for now; We can go exponential should it prove to
		// be a problem.
		time.Sleep(s.retryInterval)
	}
}

// Loop runs an infinite loop servicing synchronization requests.
func (s *Synchronizer) Loop() {
	log.Infof("Starting synchronizer loop")
	for {
		update := s.dequeue()

		log.Infof("Synchronize, type=%s", update.callbackType)

		s.SynchronizeAndRetry(context.Background(), update)

		s.complete()
	}
}

// GetModels gets the list of models.
func (s *Synchronizer) GetModels() *gnmi.Model {
	model := gnmi.NewModel(models.ModelData(),
		reflect.TypeOf((*models.Device)(nil)),
		models.SchemaTree["Device"],
		models.Unmarshal,
		//models.ΛEnum  // NOTE: There is no Enum in the aether models? So use a blank map.
		map[string]map[int64]ygot.EnumDefinition{},
	)

	return model
}

// Start the synchronizer by launching the synchronizer loop inside a thread.
func (s *Synchronizer) Start() {
	log.Infof("Synchronizer starting (postEnable=%v, postTimeout=%d, retryInterval=%s, partialUpdateEnable=%v)",
		s.postEnable,
		s.postTimeout,
		s.retryInterval,
		s.partialUpdateEnable)

	atomixClient := atomix.NewClient(atomix.WithClientID(os.Getenv("POD_NAME")))

	// TODO: Eventually we'll create a thread here that waits for config changes
	var err error
	s.sidStore, err = store.NewAtomixStore(context.Background(), atomixClient)
	if err != nil {
		log.Errorf("Can't create SID store: %v", err)
		return
	}
	go s.Loop()
}

// WithPostEnable sets the postEnable option
func WithPostEnable(postEnable bool) SynchronizerOption {
	return func(s *Synchronizer) {
		s.postEnable = postEnable
	}
}

// WithPostTimeout sets the postTimeout option
func WithPostTimeout(postTimeout time.Duration) SynchronizerOption {
	return func(s *Synchronizer) {
		s.postTimeout = postTimeout
	}
}

// WithPartialUpdateEnable sets the partialUpdateEnable option
func WithPartialUpdateEnable(partialUpdateEnable bool) SynchronizerOption {
	return func(s *Synchronizer) {
		s.partialUpdateEnable = partialUpdateEnable
	}
}

// WithTopoEndpoint specifies the onos-topo endpoint to use
func WithTopoEndpoint(topoEndpoint string) SynchronizerOption {
	return func(s *Synchronizer) {
		s.topoEndpoint = topoEndpoint
	}
}

// WithCertPaths defines certificate paths
func WithCertPaths(caPath string, keyPath string, certPath string) SynchronizerOption {
	return func(s *Synchronizer) {
		s.caPath = caPath
		s.keyPath = keyPath
		s.certPath = certPath
	}
}

// NewSynchronizer creates a new Synchronizer
func NewSynchronizer(opts ...SynchronizerOption) *Synchronizer {
	s := &Synchronizer{
		postEnable:          true,
		partialUpdateEnable: DefaultPartialUpdateEnable,
		postTimeout:         DefaultPostTimeout,
		updateChannel:       make(chan *ConfigUpdate, 1),
		retryInterval:       5 * time.Second,
		cache:               map[string]interface{}{},
		prometheus:          map[string]*metrics.Fetcher{},

		kafkaMsgChannel:   make(chan string, 10),
		kafkaErrorChannel: make(chan error, 10),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.synchronizeDeviceFunc = s.SynchronizeDevice
	return s
}
