// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

import (
	models "github.com/onosproject/config-models/models/sdn-fabric-0.1.x/api"
)

// Various typedefs to make modeling types more convenient throughout the synchronizer.

type RootDevice = models.Device                               //nolint
type DhcpServer = models.OnfDhcpServer_DhcpServer             //nolint
type Port = models.OnfSwitch_Switch_Port                      //nolint
type Route = models.OnfRoute_Route                            //nolint
type Switch = models.OnfSwitch_Switch                         //nolint
type SwitchVlan = models.OnfSwitch_Switch_Vlan                //nolint
type SwitchModel = models.OnfSwitchModel_SwitchModel          //nolint
type SwitchModelPort = models.OnfSwitchModel_SwitchModel_Port //nolint
type SwitchPortKey = models.OnfSwitch_Switch_Port_Key         //nolint

const RoleUnset = models.OnfSwitch_Switch_Role_UNSET         //nolint
const RoleUndefined = models.OnfSwitch_Switch_Role_undefined //nolint
const RoleLeaf = models.OnfSwitch_Switch_Role_leaf           //nolint
const RoleSpine = models.OnfSwitch_Switch_Role_spine         //nolint
