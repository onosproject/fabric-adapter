// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package synchronizer

import (
	"context"
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/onosproject/config-models/models/sdn-fabric-0.1.x/api"
	"github.com/onosproject/fabric-adapter/pkg/store"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var (
	deviceTestFabricID = "fabric-one"

	deviceTestDriverKey     = "driver"
	deviceTestDriverValue   = "test-driver"
	deviceTestPipeconfKey   = "pipeconf"
	deviceTestPipeconfValue = "pipe.configuration"

	deviceTestLeafID              = "leaf-one"
	deviceTestLeafDisplayName     = "Leaf 1"
	deviceTestLeafDescription     = "Test Leaf Switch 1"
	deviceTestLeafManagementIP    = "11.22.33.44"
	deviceTestLeafManagementPort  = uint16(2345)
	deviceTestSpineID             = "spine-one"
	deviceTestSpineDisplayName    = "Spine 1"
	deviceTestSpineDescription    = "Test Spine Switch 1"
	deviceTestSpineManagementIP   = "11.22.33.45"
	deviceTestSpineManagementPort = uint16(2346)
)

func newSwitch(ID *string, displayName *string, description *string,
	management *api.OnfSwitch_Switch_Management,
	attributes map[string]*api.OnfSwitch_Switch_Attribute,
	role api.E_OnfSwitch_Switch_Role) *Switch {
	onfSwitch := Switch{
		Attribute:   attributes,
		Description: description,
		DisplayName: displayName,
		Management:  management,
		Role:        role,
		SwitchId:    ID,
	}
	return &onfSwitch
}

func newAttributes() map[string]*api.OnfSwitch_Switch_Attribute {
	attributes := make(map[string]*api.OnfSwitch_Switch_Attribute)
	attributes[deviceTestDriverKey] = &api.OnfSwitch_Switch_Attribute{
		AttributeKey: &deviceTestDriverKey,
		Value:        &deviceTestDriverValue,
	}

	attributes[deviceTestPipeconfKey] = &api.OnfSwitch_Switch_Attribute{
		AttributeKey: &deviceTestPipeconfKey,
		Value:        &deviceTestPipeconfValue,
	}

	return attributes
}

func newScope(fabricID *string, onfSwitch *Switch, netconfig *OnosNetConfig) FabricScope {
	scope := FabricScope{
		NetConfig: netconfig,
		Switch:    onfSwitch,
		FabricId:  fabricID,
	}
	scope.NetConfig.Devices = make(map[string]*onosDevice)
	return scope
}

// Check the segment routing data
func checkSegmentRouting(t *testing.T, netconfDevice *onosDevice, expectedSid uint32, expectedManagementIP string, expectedIsRouter bool) {
	assert.Equal(t, expectedSid, netconfDevice.SegmentRouting.Ipv4NodeSid)
	assert.Equal(t, expectedManagementIP, netconfDevice.SegmentRouting.Ipv4Loopback)
	assert.Equal(t, expectedIsRouter, netconfDevice.SegmentRouting.IsEdgeRouter)
	expectedMac, _ := addressToMac(expectedManagementIP)
	assert.Equal(t, expectedMac, netconfDevice.SegmentRouting.RouterMac)
	assert.Empty(t, netconfDevice.SegmentRouting.AdjacencySids)
}

// Check the switch basic data
func checkBasic(t *testing.T, netconfDevice *onosDevice, displayName string) {
	assert.Equal(t, displayName, netconfDevice.Basic.Name)
	assert.Equal(t, deviceTestPipeconfValue, netconfDevice.Basic.PipeConf)
	assert.Equal(t, deviceTestDriverValue, netconfDevice.Basic.Driver)
	assert.True(t, strings.HasPrefix(netconfDevice.Basic.ManagementAddress, "grpc:"))
}

func getAtomixStore(t *testing.T) (*test.Test, store.SIDStore) {
	testAtomix := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1))
	assert.NoError(t, testAtomix.Start())

	client, err := testAtomix.NewClient("node-1")
	assert.NoError(t, err)

	sidStore, err := store.NewAtomixStore(context.Background(), client)
	assert.NoError(t, err)

	return testAtomix, sidStore
}

// TestDeviceToSegmentRouting tests conversion of a switch to a segment routing payload
func TestDeviceToSegmentRouting(t *testing.T) {
	testAtomix, sidStore := getAtomixStore(t)

	s := Synchronizer{sidStore: sidStore}

	netconfig := &OnosNetConfig{}
	management := &api.OnfSwitch_Switch_Management{
		Address:    &deviceTestLeafManagementIP,
		PortNumber: &deviceTestLeafManagementPort,
	}
	attributes := newAttributes()

	onfSwitch := newSwitch(&deviceTestLeafID, &deviceTestLeafDisplayName, &deviceTestLeafDescription, management, attributes, RoleLeaf)
	scope := newScope(&deviceTestFabricID, onfSwitch, netconfig)

	assert.NoError(t, s.handleSwitch(context.Background(), &scope))

	assert.Len(t, scope.NetConfig.Devices, 1)
	netconfDevice := scope.NetConfig.Devices["device:"+deviceTestLeafID]
	assert.NotNil(t, netconfDevice)

	// Check the segment routing data
	checkSegmentRouting(t, netconfDevice, 1, deviceTestLeafManagementIP, true)

	// check the basic data
	checkBasic(t, netconfDevice, deviceTestLeafDisplayName)

	assert.NoError(t, testAtomix.Stop())
}

// TestSIDUniqueness adds the same switch twice and makes sure that the SID of the switch does not change
func TestSIDDUniqueness(t *testing.T) {
	testAtomix, sidStore := getAtomixStore(t)

	s := Synchronizer{sidStore: sidStore}
	netconfig := &OnosNetConfig{}

	// Create a leaf switch
	management := &api.OnfSwitch_Switch_Management{
		Address:    &deviceTestLeafManagementIP,
		PortNumber: &deviceTestSpineManagementPort,
	}
	attributes := newAttributes()
	onfSwitch := newSwitch(&deviceTestLeafID, &deviceTestLeafDisplayName, &deviceTestLeafDescription, management, attributes, RoleLeaf)

	// Create a spine switch
	spineManagement := &api.OnfSwitch_Switch_Management{
		Address:    &deviceTestSpineManagementIP,
		PortNumber: &deviceTestSpineManagementPort,
	}
	spineAttributes := newAttributes()
	spineOnfSwitch := newSwitch(&deviceTestSpineID, &deviceTestSpineDisplayName, &deviceTestSpineDescription, spineManagement, spineAttributes, RoleSpine)

	// Synchronize the leaf switch
	scope := newScope(&deviceTestFabricID, onfSwitch, netconfig)
	assert.NoError(t, s.handleSwitch(context.Background(), &scope))

	// Synchronize the spine switch
	scope.Switch = spineOnfSwitch
	assert.NoError(t, s.handleSwitch(context.Background(), &scope))

	assert.Len(t, scope.NetConfig.Devices, 2)
	leafNetconfDevice := scope.NetConfig.Devices["device:"+deviceTestLeafID]
	spineNetconfDevice := scope.NetConfig.Devices["device:"+deviceTestSpineID]
	assert.NotNil(t, leafNetconfDevice)

	// Check the segment routing data
	checkSegmentRouting(t, leafNetconfDevice, 1, deviceTestLeafManagementIP, true)
	checkSegmentRouting(t, spineNetconfDevice, 2, deviceTestSpineManagementIP, false)

	// check the basic data
	checkBasic(t, leafNetconfDevice, deviceTestLeafDisplayName)
	checkBasic(t, spineNetconfDevice, deviceTestSpineDisplayName)

	// Make sure that the SIDs are different
	assert.NotEqual(t, leafNetconfDevice.SegmentRouting.Ipv4NodeSid, spineNetconfDevice.SegmentRouting.Ipv4NodeSid)

	// Now add the same switch again with an updated management address. The switch SID should not change
	spineSid := spineNetconfDevice.SegmentRouting.Ipv4NodeSid
	assert.NoError(t, s.handleSwitch(context.Background(), &scope))
	spineNetconfDeviceAfter := scope.NetConfig.Devices["device:"+deviceTestSpineID]
	checkSegmentRouting(t, spineNetconfDeviceAfter, spineSid, deviceTestSpineManagementIP, false)

	assert.Equal(t, spineSid, spineNetconfDeviceAfter.SegmentRouting.Ipv4NodeSid)
	assert.Contains(t, spineNetconfDeviceAfter.Basic.ManagementAddress, deviceTestSpineManagementIP)

	assert.NoError(t, testAtomix.Stop())
}
