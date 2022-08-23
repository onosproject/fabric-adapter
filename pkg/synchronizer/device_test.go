// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package synchronizer

import (
	"context"
	"fmt"
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

func newVlan(vlanNumber uint16, subnet string, description string) *api.OnfSwitch_Switch_Vlan {
	vlDisplayName := fmt.Sprintf("Test Vlan %s-%d", description, vlanNumber)
	vlDescription := fmt.Sprintf("TestVlan%s-%d", description, vlanNumber)
	vlSubnets := []string{subnet}
	vl := &api.OnfSwitch_Switch_Vlan{
		Description: &vlDescription,
		DisplayName: &vlDisplayName,
		Subnet:      vlSubnets,
		VlanId:      &vlanNumber,
	}
	return vl
}

func addNewPort(sw *Switch,
	portKey api.OnfSwitch_Switch_Port_Key,
	cageNumber uint8, channelNumber uint8,
	portDescription string, portDisplayName string,
	portVlans *api.OnfSwitch_Switch_Port_Vlans) {
	port := &api.OnfSwitch_Switch_Port{
		CageNumber:       &cageNumber,
		ChannelNumber:    &channelNumber,
		Description:      &portDescription,
		DhcpConnectPoint: nil,
		DisplayName:      &portDisplayName,
		Speed:            0,
		State:            nil,
		Vlans:            portVlans,
	}
	sw.Port[portKey] = port
}

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
		Port:        make(map[api.OnfSwitch_Switch_Port_Key]*api.OnfSwitch_Switch_Port),
		Vlan:        make(map[uint16]*api.OnfSwitch_Switch_Vlan),
	}

	vlTaggedID := uint16(44)
	vlTagged := newVlan(vlTaggedID, "11.22.33.44/24", "tagged")

	vlUntaggedID := uint16(55)
	vlUntagged := newVlan(vlUntaggedID, "11.22.33.55/24", "untagged")

	vlans := make(map[uint16]*api.OnfSwitch_Switch_Vlan)
	vlans[*vlTagged.VlanId] = vlTagged
	vlans[*vlUntagged.VlanId] = vlUntagged
	onfSwitch.Vlan = vlans

	cageNumber := uint8(2)
	channelNumber := uint8(2)
	portDescription := "port1"
	portDisplayName := "Port 1"

	portKey := api.OnfSwitch_Switch_Port_Key{
		CageNumber:    cageNumber,
		ChannelNumber: channelNumber,
	}

	taggedVlanIds := []uint16{vlTaggedID}
	portVlans := &api.OnfSwitch_Switch_Port_Vlans{
		Tagged:   taggedVlanIds,
		Untagged: &vlUntaggedID,
	}
	addNewPort(&onfSwitch, portKey,
		cageNumber,
		channelNumber,
		portDescription,
		portDisplayName,
		portVlans)
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

func newTestModel() *SwitchModel {
	description := "Model for testing"
	displayName := "TestModel"
	id := "test"
	port := make(map[uint8]*api.OnfSwitchModel_SwitchModel_Port)
	for i := uint8(1); i <= 16; i++ {
		portDescription := fmt.Sprintf("Port %d", i)
		port[i] = &api.OnfSwitchModel_SwitchModel_Port{
			CageNumber:  &i,
			Description: &portDescription,
			DisplayName: &portDescription,
		}
	}
	model := &SwitchModel{
		Attribute:     nil,
		Description:   &description,
		DisplayName:   &displayName,
		Port:          port,
		SwitchModelId: &id,
	}
	return model
}

func newScope(fabricID *string, onfSwitch *Switch, netconfig *OnosNetConfig) FabricScope {
	scope := FabricScope{
		NetConfig: netconfig,
		Switch:    onfSwitch,
		FabricId:  fabricID,
	}
	scope.NetConfig.Devices = make(map[string]*onosDevice)
	scope.NetConfig.Ports = make(map[string]*onosPort)
	scope.SwitchModel = newTestModel()

	return scope
}

// Check the segment routing data
func checkSegmentRoutingDevice(t *testing.T, netconfDevice *onosDevice, expectedSid uint32, expectedManagementIP string, expectedIsRouter bool) {
	assert.Equal(t, expectedSid, netconfDevice.SegmentRouting.Ipv4NodeSid)
	assert.Equal(t, expectedManagementIP, netconfDevice.SegmentRouting.Ipv4Loopback)
	assert.Equal(t, expectedIsRouter, netconfDevice.SegmentRouting.IsEdgeRouter)
	expectedMac, _ := addressToMac(expectedManagementIP)
	assert.Equal(t, expectedMac, netconfDevice.SegmentRouting.RouterMac)
	assert.Empty(t, netconfDevice.SegmentRouting.AdjacencySids)
}

func checkSegmentRoutingPort(t *testing.T, netconf *onosPort, taggedVlan uint16, untaggedVlan uint16, expectedSubnets []string) {
	assert.Len(t, netconf.Interfaces, 1)
	assert.Len(t, netconf.Interfaces[0].Ips, len(expectedSubnets))
	assert.Equal(t, untaggedVlan, netconf.Interfaces[0].VlanUntagged)
	assert.Equal(t, taggedVlan, netconf.Interfaces[0].VlanTagged[0])
	for _, expectedSubnet := range expectedSubnets {
		found := false
		for _, subnet := range netconf.Interfaces[0].Ips {
			if subnet == expectedSubnet {
				found = true
				break
			}
		}
		assert.Truef(t, found, "Didn't find expected subnet %s", expectedSubnet)
	}
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
	checkSegmentRoutingDevice(t, netconfDevice, 1, deviceTestLeafManagementIP, true)

	port, ok := scope.NetConfig.Ports["device:leaf-one/202"]
	assert.True(t, ok)
	expectedSubnets := []string{"11.22.33.55/24", "11.22.33.44/24"}

	checkSegmentRoutingPort(t, port, 44, 55, expectedSubnets)

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
	checkSegmentRoutingDevice(t, leafNetconfDevice, 1, deviceTestLeafManagementIP, true)
	checkSegmentRoutingDevice(t, spineNetconfDevice, 2, deviceTestSpineManagementIP, false)

	// check the basic data
	checkBasic(t, leafNetconfDevice, deviceTestLeafDisplayName)
	checkBasic(t, spineNetconfDevice, deviceTestSpineDisplayName)

	// Make sure that the SIDs are different
	assert.NotEqual(t, leafNetconfDevice.SegmentRouting.Ipv4NodeSid, spineNetconfDevice.SegmentRouting.Ipv4NodeSid)

	// Now add the same switch again with an updated management address. The switch SID should not change
	spineSid := spineNetconfDevice.SegmentRouting.Ipv4NodeSid
	assert.NoError(t, s.handleSwitch(context.Background(), &scope))
	spineNetconfDeviceAfter := scope.NetConfig.Devices["device:"+deviceTestSpineID]
	checkSegmentRoutingDevice(t, spineNetconfDeviceAfter, spineSid, deviceTestSpineManagementIP, false)

	assert.Equal(t, spineSid, spineNetconfDeviceAfter.SegmentRouting.Ipv4NodeSid)
	assert.Contains(t, spineNetconfDeviceAfter.Basic.ManagementAddress, deviceTestSpineManagementIP)

	assert.NoError(t, testAtomix.Stop())
}
