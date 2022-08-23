// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package synchronizer

import (
	"bytes"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/config-models/models/sdn-fabric-0.1.x/api"
	"github.com/onosproject/fabric-adapter/pkg/stratum_hal"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStratumSwitch(t *testing.T) {
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

	assert.NoError(t, s.handleStratumSwitch(&scope))

	//singleton_ports {
	//	id: 202
	//	name: "2/2"
	//	slot: 1
	//	port: 2
	//	channel: 3
	//	speed_bps: 10000000000 # 10G
	//	config_params {
	//	admin_state: ADMIN_STATE_ENABLED
	//	autoneg: TRI_STATE_TRUE
	//}
	//	node: 1
	//}
	assert.Len(t, scope.StratumChassisConfig.SingletonPorts, 1)
	port202 := scope.StratumChassisConfig.SingletonPorts[0]
	assert.Equal(t, uint32(202), port202.Id)
	assert.Equal(t, "Port 2/2", port202.Name)
	assert.Equal(t, int32(1), port202.Slot)
	assert.Equal(t, int32(2), port202.Port)
	//assert.Equal(t, 3, port202.Channel)
	//assert.Equal(t, uint64(10000000000), port202.SpeedBps)
	assert.Equal(t, stratum_hal.AdminState_ADMIN_STATE_ENABLED, port202.ConfigParams.AdminState)
	//assert.Equal(t, stratum_hal.TriState_TRI_STATE_TRUE, port202.ConfigParams.Autoneg)
	assert.Equal(t, uint64(1), port202.Node)

	assert.NoError(t, testAtomix.Stop())
}

func TestStratumFormatting(t *testing.T) {
	node1 := stratum_hal.Node{
		Id:    1,
		Slot:  1,
		Index: 1,
	}
	configParams1 := &stratum_hal.PortConfigParams{
		AdminState: stratum_hal.AdminState_ADMIN_STATE_ENABLED,
		Autoneg:    stratum_hal.TriState_TRI_STATE_FALSE,
	}
	configParams200 := &stratum_hal.PortConfigParams{
		AdminState: stratum_hal.AdminState_ADMIN_STATE_ENABLED,
		Autoneg:    stratum_hal.TriState_TRI_STATE_TRUE,
	}
	singletonPort1 := &stratum_hal.SingletonPort{
		Id:           1,
		Name:         "1/0",
		Slot:         1,
		Port:         1,
		SpeedBps:     100000000000,
		Node:         1,
		ConfigParams: configParams1,
	}
	singletonPort200 := &stratum_hal.SingletonPort{
		Id:           200,
		Name:         "2/0",
		Slot:         1,
		Port:         1,
		SpeedBps:     10000000000,
		Node:         1,
		ConfigParams: configParams200,
	}
	singletonPort201 := &stratum_hal.SingletonPort{
		Id:           201,
		Name:         "2/1",
		Slot:         1,
		Port:         2,
		Channel:      2,
		SpeedBps:     10000000000,
		Node:         1,
		ConfigParams: configParams200,
	}
	singletonPort202 := &stratum_hal.SingletonPort{
		Id:           202,
		Name:         "2/2",
		Slot:         1,
		Port:         2,
		Channel:      3,
		SpeedBps:     10000000000,
		Node:         1,
		ConfigParams: configParams200,
	}
	singletonPort203 := &stratum_hal.SingletonPort{
		Id:           203,
		Name:         "2/3",
		Slot:         1,
		Port:         2,
		Channel:      4,
		SpeedBps:     10000000000,
		Node:         1,
		ConfigParams: configParams200,
	}
	chassisConfig := &stratum_hal.ChassisConfig{
		Description: "Chassis config example",
		Chassis: &stratum_hal.Chassis{
			Platform: stratum_hal.Platform_PLT_GENERIC_BAREFOOT_TOFINO,
			Name:     "leaf-1",
		},
		Nodes:                    []*stratum_hal.Node{&node1},
		SingletonPorts:           []*stratum_hal.SingletonPort{singletonPort1, singletonPort200, singletonPort201, singletonPort202, singletonPort203},
		TrunkPorts:               nil,
		PortGroups:               nil,
		VendorConfig:             nil,
		OpticalNetworkInterfaces: nil,
	}

	var protoStringBytes bytes.Buffer
	err := proto.MarshalText(&protoStringBytes, chassisConfig)
	assert.NoError(t, err)

	protoString := protoStringBytes.String()
	assert.NotNil(t, protoString)
}
