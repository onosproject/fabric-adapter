// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package synchronizer

import (
	"github.com/onosproject/config-models/models/sdn-fabric-0.1.x/api"
	"github.com/onosproject/fabric-adapter/pkg/stratum_hal"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func checkStratumPort(t *testing.T, singletonPort *stratum_hal.SingletonPort, ID uint32, name string, port int32, channel int32, speedBps uint64, autonegState stratum_hal.TriState) {
	assert.Equal(t, ID, singletonPort.Id)
	assert.Equal(t, name, singletonPort.Name)
	assert.Equal(t, int32(1), singletonPort.Slot)
	assert.Equal(t, port, singletonPort.Port)
	assert.Equal(t, channel, singletonPort.Channel)
	assert.Equal(t, speedBps, singletonPort.SpeedBps)
	assert.Equal(t, stratum_hal.AdminState_ADMIN_STATE_ENABLED, singletonPort.ConfigParams.AdminState)
	assert.Equal(t, autonegState, singletonPort.ConfigParams.Autoneg)
	assert.Equal(t, uint64(1), singletonPort.Node)
}

func findPort(t *testing.T, portID uint32, scope FabricScope) *stratum_hal.SingletonPort {
	for _, port := range scope.StratumChassisConfig.SingletonPorts {
		if port.Id == portID {
			return port
		}
	}

	assert.Failf(t, "Could not find port", "could not find port %d", portID)
	return nil
}

func TestStratumSwitch(t *testing.T) {
	// These test cases come from https://docs.sd-fabric.org/sdfabric-1.1/configuration/chassis.html#singleton-port
	const (
		cageNumber10      = uint8(1)
		portDescription10 = "port cage 1"
		portDisplayName10 = "Port 1/0"

		cageNumber20      = uint8(2)
		channelNumber20   = uint8(1)
		portDescription20 = "Port cage 2/channel 0"
		portDisplayName20 = "Port 2/0"

		cageNumber23      = uint8(2)
		channelNumber23   = uint8(3)
		portDescription23 = "Port cage 2/channel 3"
		portDisplayName23 = "Port 2/3"

		gig        = 10e8
		tenGig     = 10e9
		hundredGig = 10e10
	)
	testAtomix, sidStore := getAtomixStore(t)

	s := Synchronizer{sidStore: sidStore}

	netconfig := &OnosNetConfig{}
	management := &api.OnfSwitch_Switch_Management{
		Address:    &deviceTestLeafManagementIP,
		PortNumber: &deviceTestLeafManagementPort,
	}
	attributes := newAttributes()

	onfSwitch := newSwitch(&deviceTestLeafID, &deviceTestLeafDisplayName, &deviceTestLeafDescription, management, attributes, RoleLeaf)

	portKey23 := api.OnfSwitch_Switch_Port_Key{
		CageNumber:    cageNumber23,
		ChannelNumber: channelNumber23,
	}

	addNewPort(onfSwitch, portKey23,
		cageNumber23,
		channelNumber23,
		portDescription23,
		portDisplayName23,
		api.OnfSdnFabricTypes_Speed_speed_autoneg)

	portKey10 := api.OnfSwitch_Switch_Port_Key{
		CageNumber: cageNumber10,
	}

	addNewPort(onfSwitch, portKey10,
		cageNumber10,
		0,
		portDescription10,
		portDisplayName10,
		api.OnfSdnFabricTypes_Speed_speed_100g)

	portKey20 := api.OnfSwitch_Switch_Port_Key{
		CageNumber:    cageNumber20,
		ChannelNumber: channelNumber20,
	}

	addNewPort(onfSwitch, portKey20,
		cageNumber20,
		channelNumber20,
		portDescription20,
		portDisplayName20,
		api.OnfSdnFabricTypes_Speed_speed_10g)

	scope := newScope(&deviceTestFabricID, onfSwitch, netconfig)

	assert.NoError(t, s.handleStratumSwitch(&scope))

	assert.Len(t, scope.StratumChassisConfig.SingletonPorts, 3)

	port202 := findPort(t, 202, scope)
	checkStratumPort(t, port202, 202, "Port 2/2", int32(cageNumber23), int32(channelNumber23), tenGig, stratum_hal.TriState_TRI_STATE_TRUE)

	port1 := findPort(t, 1, scope)
	checkStratumPort(t, port1, 1, "Port 1/0", int32(cageNumber10), 0, hundredGig, stratum_hal.TriState_TRI_STATE_FALSE)

	port200 := findPort(t, 200, scope)
	checkStratumPort(t, port200, 200, "Port 2/0", int32(cageNumber20), int32(channelNumber20), tenGig, stratum_hal.TriState_TRI_STATE_FALSE)

	assert.NoError(t, testAtomix.Stop())
}

func TestGNMI(t *testing.T) {
	cl := testClient{expectedStatus: http.StatusOK}
	gnmiPushClient = &cl
	testAtomix, sidStore := getAtomixStore(t)

	s := Synchronizer{sidStore: sidStore}

	netconfig := &OnosNetConfig{}
	management := &api.OnfSwitch_Switch_Management{
		Address:    &deviceTestLeafManagementIP,
		PortNumber: &deviceTestLeafManagementPort,
	}
	attributes := newAttributes()
	onfSwitch := newSwitch(&deviceTestLeafID, &deviceTestLeafDisplayName, &deviceTestLeafDescription, management, attributes, RoleLeaf)

	portKey := api.OnfSwitch_Switch_Port_Key{
		CageNumber:    5,
		ChannelNumber: 9,
	}

	addNewPort(onfSwitch, portKey,
		5,
		9,
		"port description",
		"port display name",
		api.OnfSdnFabricTypes_Speed_speed_100g)

	scope := newScope(&deviceTestFabricID, onfSwitch, netconfig)
	sw := make(map[string]*api.OnfSwitch_Switch)
	onfSwitch.ModelId = scope.SwitchModel.SwitchModelId
	sw[*onfSwitch.SwitchId] = onfSwitch
	scope.Fabric = &RootDevice{
		Switch: sw,
	}
	switchModelID := "test"
	switchModel := SwitchModel{
		Attribute:     nil,
		Description:   nil,
		DisplayName:   nil,
		Pipeline:      0,
		Port:          nil,
		SwitchModelId: &switchModelID,
	}
	scope.Fabric.Switch = sw
	scope.Fabric.SwitchModel = make(map[string]*api.OnfSwitchModel_SwitchModel)
	scope.Fabric.SwitchModel["test"] = &switchModel
	errorCount, err := s.SynchronizeFabricToStratum(&scope)
	assert.NoError(t, err)
	assert.Equal(t, 0, errorCount)

	assert.NoError(t, testAtomix.Stop())
}
