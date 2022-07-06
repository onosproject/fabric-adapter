// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

import (
	"fmt"
)

// Functions here to ease in looking things up

func lookupSwitchModel(scope *FabricScope, id *string) (*SwitchModel, error) {
	if (id == nil) || (*id == "") {
		return nil, fmt.Errorf("SwitchModel id is blank")
	}
	swm, okay := scope.Fabric.SwitchModel[*id]
	if !okay {
		return nil, fmt.Errorf("SwitchModel %s not found", *id)
	}
	return swm, nil
}

func lookupSwitchModelPort(model *SwitchModel, cage *uint8) (*SwitchModelPort, error) {
	port, okay := model.Port[*cage]
	if !okay {
		return nil, fmt.Errorf("SwitchModel has no port matching %v", cage)
	}

	return port, nil
}

func lookupSwitchVlan(sw *Switch, id *uint16) (*SwitchVlan, error) {
	if id == nil {
		return nil, fmt.Errorf("Vlan id is blank")
	}
	vlan, okay := sw.Vlan[*id]
	if !okay {
		return nil, fmt.Errorf("Switch Vlan %v not found", *id)
	}
	return vlan, nil
}
