// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/binary"
	"github.com/atomix/atomix-go-client/pkg/atomix/counter"
	atomixerrors "github.com/atomix/atomix-go-framework/pkg/atomix/errors"
	"io"
	"time"

	"github.com/atomix/atomix-go-client/pkg/atomix"
	_map "github.com/atomix/atomix-go-client/pkg/atomix/map"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger()

const (
	// SidCounter is the name used for atomix counter for generating unique SIDs
	SidCounter = "fabric-adapter-sid-counter"

	// SidMap is the name used for atomix counter for generating unique SIDs
	SidMap = "fabric-adapter-sid-map"
)

// NewAtomixStore returns a new persistent Store
func NewAtomixStore(ctx context.Context, atomixClient atomix.Client) (SIDStore, error) {

	nextSID, err := atomixClient.GetCounter(ctx, SidCounter)
	if err != nil {
		log.Warnf("Error creating atomix counter: %v", err)
		return nil, err
	}
	startingValue, err := nextSID.Get(ctx)
	if err != nil {
		log.Warnf("Error querying atomix counter: %v", err)
		return nil, err
	}
	if startingValue < 100 {
		// Reserve the first 100 SIDs for segment routing
		err = nextSID.Set(ctx, 100)
		if err != nil {
			log.Warnf("Error initializing atomix counter: %v", err)
			return nil, err
		}
	}
	sidMap, err := atomixClient.GetMap(ctx, SidMap)
	if err != nil {
		log.Warnf("Error creating atomix map: %v", err)
		return nil, err
	}

	store := &SIDAtomixStore{
		nextSID: nextSID,
		sidMap:  sidMap,
	}

	return store, nil
}

// SIDStore stores UE information
type SIDStore interface {
	io.Closer

	// Get a new SID for the given switch
	Get(ctx context.Context, switchID string) (uint32, error)
}

// SIDAtomixStore is the object implementation of the Store
type SIDAtomixStore struct {
	nextSID counter.Counter
	sidMap  _map.Map
}

func uint32ToBytes(i uint32) []byte {
	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, i)

	return value
}

func bytesToUint32(value []byte) uint32 {
	return binary.LittleEndian.Uint32(value)
}

// Get gets the SID assigned to the given switch, creating a new one if necessary
func (s *SIDAtomixStore) Get(ctx context.Context, switchID string) (uint32, error) {
	if switchID == "" {
		return 0, errors.NewInvalid("ID cannot be empty")
	}

	log.Infof("Looking for switch %s", switchID)
	entry, err := s.sidMap.Get(ctx, switchID)
	if entry != nil && err == nil {
		sid := bytesToUint32(entry.Value)
		log.Infof("Switch found with SID %d", sid)
		return sid, nil
	}

	if !atomixerrors.IsNotFound(err) {
		log.Errorf("Error getting from SID map: %v", err)
		return 0, err
	}

	newSid, err := s.nextSID.Increment(ctx, 1)
	if err == nil {
		log.Infof("Allocated new SID %d", newSid)
		sidValue := uint32ToBytes(uint32(newSid))

		_, err = s.sidMap.Put(ctx, switchID, sidValue)
	}
	return uint32(newSid), err
}

// Close closes the store
func (s *SIDAtomixStore) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.nextSID.Close(ctx)
	if err != nil {
		return err
	}
	err = s.sidMap.Close(ctx)
	if err != nil {
		return err
	}
	return nil
}
