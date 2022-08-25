// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/config-models/models/sdn-fabric-0.1.x/api"
	"github.com/onosproject/fabric-adapter/pkg/stratum_hal"
	"github.com/onosproject/sdcore-adapter/pkg/gnmi"
	"github.com/pkg/errors"
	"sort"
	"time"
)

func (s *Synchronizer) handleSwitchPort(scope *FabricScope, p *Port) error {
	sw := scope.Switch
	model := scope.SwitchModel

	modelPort, err := lookupSwitchModelPort(model, p.CageNumber)
	if err != nil {
		return err
	}

	portID := switchCageChannelToDeviceId(sw, p.CageNumber, p.ChannelNumber)

	iface := onosInterface{
		Name: *p.DisplayName,
		Ips:  []string{},
	}

	if p.Vlans != nil {
		if p.Vlans.Untagged != nil {
			vlan, err := lookupSwitchVlan(sw, p.Vlans.Untagged)
			if err != nil {
				return err
			}
			iface.VlanUntagged = *p.Vlans.Untagged
			iface.Ips = append(iface.Ips, vlan.Subnet...)
		}
		for _, vlanID := range p.Vlans.Tagged {
			vlan, err := lookupSwitchVlan(sw, &vlanID)
			if err != nil {
				return err
			}
			iface.VlanTagged = append(iface.VlanTagged, vlanID)
			iface.Ips = append(iface.Ips, vlan.Subnet...)
		}
	}

	port := &onosPort{
		Interfaces: []onosInterface{iface},
	}

	scope.NetConfig.Ports[portID] = port

	_ = model
	_ = modelPort

	return nil
}

func (s *Synchronizer) handleStratumSwitchPort(scope *FabricScope, p *Port) error {
	model := scope.SwitchModel

	// Make sure the port is in the model
	_, err := lookupSwitchModelPort(model, p.CageNumber)
	if err != nil {
		return err
	}

	// Determine port speed
	var autoneg = stratum_hal.TriState_TRI_STATE_FALSE
	var speedBPS uint64
	const gig = 10e8
	switch p.Speed {
	case api.OnfSdnFabricTypes_Speed_speed_100g:
		speedBPS = 100 * gig
	case api.OnfSdnFabricTypes_Speed_speed_10g:
		speedBPS = 10 * gig
	case api.OnfSdnFabricTypes_Speed_speed_1g:
		speedBPS = gig
	case api.OnfSdnFabricTypes_Speed_speed_2_5g:
		speedBPS = 2.5 * gig
	case api.OnfSdnFabricTypes_Speed_speed_25g:
		speedBPS = 25 * gig
	case api.OnfSdnFabricTypes_Speed_speed_400g:
		speedBPS = 400 * gig
	case api.OnfSdnFabricTypes_Speed_speed_40g:
		speedBPS = 40 * gig
	case api.OnfSdnFabricTypes_Speed_speed_5g:
		speedBPS = 5 * gig
	case api.OnfSdnFabricTypes_Speed_speed_autoneg:
		autoneg = stratum_hal.TriState_TRI_STATE_TRUE
		speedBPS = 10 * gig // TODO make this an attribute
	}

	// port configuration parameters
	configParams := &stratum_hal.PortConfigParams{
		AdminState: stratum_hal.AdminState_ADMIN_STATE_ENABLED,
		Autoneg:    autoneg,
	}

	// determine the id, slot, port and channel for the stratum model
	slot := int32(1) // TODO if the switch has more than one line card, this will be wrong for all but the first card
	port := int32(*p.CageNumber)
	channel := uint32(*p.ChannelNumber)
	var id uint32
	var name string

	if channel != 0 {
		id = (uint32(port) * 100) + (channel - 1)
		name = fmt.Sprintf("Port %d/%d", port, channel-1)
	} else {
		id = uint32(port)
		name = fmt.Sprintf("Port %d/0", port)
	}

	// Add the port to the stratum config
	singletonPort := &stratum_hal.SingletonPort{
		Id:           id,
		Name:         name,
		Slot:         slot,
		Port:         port,
		Channel:      int32(channel),
		SpeedBps:     speedBPS,
		Node:         1,
		ConfigParams: configParams,
	}
	scope.StratumChassisConfig.SingletonPorts = append(scope.StratumChassisConfig.SingletonPorts, singletonPort)

	return nil
}

func (s *Synchronizer) handleSwitch(ctx context.Context, scope *FabricScope) error {
	var err error

	sw := scope.Switch

	log.Infof("Fabric %s handling switch %s", *scope.FabricId, *sw.SwitchId)

	if sw.Management == nil || sw.Management.Address == nil || sw.Management.PortNumber == nil {
		return fmt.Errorf("fabric %s switch %s has no management address", *scope.FabricId, *sw.SwitchId)
	}

	device := &onosDevice{}

	device.Basic.Name = *sw.DisplayName
	driver := sw.Attribute["driver"]
	if driver == nil || driver.Value == nil || *driver.Value == "" {
		return errors.New("switch driver attribute must be specified")
	}

	device.Basic.Driver = *driver.Value
	device.SegmentRouting.Ipv4NodeSid, err = s.sidStore.Get(ctx, *sw.SwitchId)
	if err != nil {
		return fmt.Errorf("fabric %s switch %s unable to create SID: %s", *scope.FabricId, *sw.SwitchId, err)
	}
	device.SegmentRouting.IsEdgeRouter = sw.Role != RoleSpine
	device.SegmentRouting.Ipv4Loopback = *sw.Management.Address
	device.SegmentRouting.RouterMac, err = addressToMac(*sw.Management.Address)
	if err != nil {
		return fmt.Errorf("fabric %s switch %s unable to create MAC: %s", *scope.FabricId, *sw.SwitchId, err)
	}

	pipeconf := sw.Attribute["pipeconf"]
	if pipeconf == nil || pipeconf.Value == nil || *pipeconf.Value == "" {
		return errors.New("switch pipeconf attribute must be specified")
	}
	device.Basic.PipeConf = *pipeconf.Value
	device.Basic.ManagementAddress = fmt.Sprintf("grpc://%s:%d?device_id=1", *sw.Management.Address, *sw.Management.PortNumber)
	// omit for now: locType, gridX, gridY

	// segmentRouting
	// Ipv4 Node Sid, Ipv4 Loopback, Router Mac, Is Edge Router, Adjacency Sids
	device.SegmentRouting.AdjacencySids = make([]uint16, 0)
	device.SegmentRouting.Ipv4Loopback = managementAddressToIP(*sw.Management.Address)
	device.SegmentRouting.IsEdgeRouter = sw.Role != RoleSpine
	device.SegmentRouting.RouterMac, err = addressToMac(device.SegmentRouting.Ipv4Loopback)
	if err != nil {
		return fmt.Errorf("fabric %s switch %s unable to create routermac: %s", *scope.FabricId, *sw.SwitchId, err)
	}

	scope.NetConfig.Devices["device:"+*sw.SwitchId] = device

	// Ports

	for _, port := range sw.Port {
		err := s.handleSwitchPort(scope, port)
		if err != nil {
			// log the error and continue with next port
			log.Warn(err)
		}
	}

	// Pairing

	if (sw.SwitchPair != nil) && (sw.SwitchPair.PairedSwitch != nil) {
		if (sw.SwitchPair.PairingPort == nil) || (len(sw.SwitchPair.PairingPort) == 0) {
			log.Warnf("Switch %s has PairedSwitch but no PairingPorts", *sw.SwitchId)
		} else if len(sw.SwitchPair.PairingPort) > 1 {
			// limitation for now, only 1 pairing port
			log.Warnf("Switch %s has PairedSwitch and has more than one PairingPort", *sw.SwitchId)
		} else {
			device.SegmentRouting.PairDeviceID = *sw.SwitchId

			for _, pairingPort := range sw.SwitchPair.PairingPort {
				device.SegmentRouting.PairLocalPort = cageChannelToPort(pairingPort.CageNumber, pairingPort.ChannelNumber)
			}
		}
	}

	return nil
}

func (s *Synchronizer) handleStratumSwitch(scope *FabricScope) error {
	sw := scope.Switch

	log.Infof("Fabric %s handling switch %s for  stratum", *scope.FabricId, *sw.SwitchId)

	if sw.Management == nil || sw.Management.Address == nil || sw.Management.PortNumber == nil {
		return fmt.Errorf("fabric %s switch %s has no management address", *scope.FabricId, *sw.SwitchId)
	}

	scope.StratumChassisConfig.Description = *sw.DisplayName

	node := stratum_hal.Node{ // TODO is this right?
		Id:    1,
		Slot:  1,
		Index: 1,
	}
	scope.StratumChassisConfig.Nodes = []*stratum_hal.Node{&node}

	scope.StratumChassisConfig.Chassis = &stratum_hal.Chassis{
		Platform: stratum_hal.Platform_PLT_GENERIC_BAREFOOT_TOFINO,
		Name:     *sw.DisplayName,
	}

	// Ports

	for _, port := range sw.Port {
		err := s.handleStratumSwitchPort(scope, port)
		if err != nil {
			// log the error and continue with next port
			log.Warn(err)
		}
	}

	return nil
}

func (s *Synchronizer) handleRoute(scope *FabricScope, route *Route) error {
	err := validateRoute(route)
	if err != nil {
		return err
	}

	oRoute := onosRoute{
		Prefix:  *route.Prefix,
		NextHop: *route.Address,
	}

	routeApp, okay := scope.NetConfig.Apps[onosRouteAppName]
	if !okay {
		routeApp = &onosApp{
			Routes: []onosRoute{},
		}
		scope.NetConfig.Apps[onosRouteAppName] = routeApp
	}

	routeApp.Routes = append(routeApp.Routes, oRoute)

	return nil
}

// SynchronizeFabricToOnos pushes a fabric to an onos netconfig
func (s *Synchronizer) SynchronizeFabricToOnos(ctx context.Context, scope *FabricScope) (int, error) {
	// be deterministic...
	switchIDKeys := []string{}
	for k := range scope.Fabric.Switch {
		switchIDKeys = append(switchIDKeys, k)
	}
	sort.Strings(switchIDKeys)

nextSwitch:
	for _, k := range switchIDKeys {
		var err error
		scope.Switch = scope.Fabric.Switch[k]
		scope.SwitchModel, err = lookupSwitchModel(scope, scope.Switch.ModelId)
		if err != nil {
			// log the error and continue with next switch
			log.Warn(err)
			continue nextSwitch
		}

		err = s.handleSwitch(ctx, scope)
		if err != nil {
			// log the error and continue with next switch
			log.Warn(err)
		}
	}

	for _, route := range scope.Fabric.Route {
		err := s.handleRoute(scope, route)
		if err != nil {
			// log the error and continue with next route
			log.Warn(err)
		}
	}

	if s.partialUpdateEnable && s.CacheCheck(CacheModelNetConfig, *scope.FabricId, scope.NetConfig) {
		log.Infof("Fabric %s netconfig has not changed", *scope.FabricId)
		return 0, nil
	}

	data, err := json.MarshalIndent(scope.NetConfig, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("Fabric %s failed to Marshal netconfig Json: %s", *scope.FabricId, err)
	}

	if scope.OnosEndpoint == nil {
		return 0, fmt.Errorf("Fabric %s has no netconfig endpoint to push to", *scope.FabricId)
	}

	url := fmt.Sprintf("%sonos/v1/network/configuration", *scope.OnosEndpoint)
	restPusher := NewRestPusher(url, *scope.OnosUsername, *scope.OnosPassword, data)
	err = restPusher.PushUpdate()
	if err != nil {
		return 1, fmt.Errorf("Fabric %s failed to Push netconfig update: %s", *scope.FabricId, err)
	}

	s.CacheUpdate(CacheModelNetConfig, *scope.FabricId, scope.NetConfig)

	return 0, nil
}

// SynchronizeFabricToStratum pushes a fabric to stratum switches
func (s *Synchronizer) SynchronizeFabricToStratum(scope *FabricScope) (int, error) {
	// be deterministic...
	switchIDKeys := []string{}
	for k := range scope.Fabric.Switch {
		switchIDKeys = append(switchIDKeys, k)
	}
	sort.Strings(switchIDKeys)

nextSwitch:
	for _, k := range switchIDKeys {
		var err error
		scope.StratumChassisConfig = stratum_hal.ChassisConfig{}
		scope.Switch = scope.Fabric.Switch[k]
		scope.SwitchModel, err = lookupSwitchModel(scope, scope.Switch.ModelId)
		if err != nil {
			// log the error and continue with next switch
			log.Warn(err)
			continue nextSwitch
		}

		err = s.handleStratumSwitch(scope)
		if err != nil {
			// log the error and continue with next switch
			log.Warn(err)
		}

		var protoStringBytes bytes.Buffer
		err = proto.MarshalText(&protoStringBytes, &scope.StratumChassisConfig)
		if err != nil {
			return 1, err
		}
		protoString := protoStringBytes.String()
		log.Warnf("proto string for switch %s is:\n%s\n", *scope.Switch.SwitchId, protoString)

		// Push proto here
		gnmiPusher := NewGNMIPusher("/", "stratum", protoString)
		err = gnmiPusher.PushUpdate()

		if err != nil {
			return 1, err
		}
	}

	return 0, nil
}

// SynchronizeDevice synchronizes a device. Two sets of error state are returned:
//   1) pushFailures -- a count of pushes that failed to the core. Synchronizer should retry again later.
//   2) error -- a fatal error that occurred during synchronization.
func (s *Synchronizer) SynchronizeDevice(ctx context.Context, allConfig *gnmi.ConfigForest) (int, error) {
	pushFailuresTotal := 0
	for fabricID, fabricConfig := range allConfig.Configs {
		device := fabricConfig.(*RootDevice)

		log.Info("SynchronizeDevce")

		controllerInfo, err := lookupFabricControllerInfo(ctx, s, fabricID)
		if err != nil {
			return 0, err
		}

		tStart := time.Now()
		KpiSynchronizationTotal.WithLabelValues(fabricID).Inc()

		uri := fmt.Sprintf("http://%s:%d/", controllerInfo.ControlEndpoint.Address, controllerInfo.ControlEndpoint.Port)
		log.Info("controller uri: %s", uri)
		scope := &FabricScope{
			FabricId:        &fabricID,
			Fabric:          device,
			OnosEndpoint:    aStr(uri),
			OnosUsername:    aStr(controllerInfo.Username),
			OnosPassword:    aStr(controllerInfo.Password),
			StratumEndpoint: aStr(uri),
			NetConfig: &OnosNetConfig{
				Devices: map[string]*onosDevice{},
				Ports:   map[string]*onosPort{},
				Apps:    map[string]*onosApp{},
			}}

		pushFailures, err := s.SynchronizeFabricToOnos(ctx, scope)
		if err != nil {
			log.Warnf("Failed to push fabric to ONOS %s: %v", fabricID, err)
		}

		pushStratumFailures, err := s.SynchronizeFabricToStratum(scope)
		if err != nil {
			log.Warnf("Failed to push fabric to ONOS %s: %v", fabricID, err)
		}

		pushFailuresTotal += pushFailures
		pushFailuresTotal += pushStratumFailures
		KpiSynchronizationDuration.WithLabelValues(fabricID).Observe(time.Since(tStart).Seconds())
	}

	return pushFailuresTotal, nil
}
