// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

import (
	"context"
	"encoding/json"
	"fmt"
	atomixerrors "github.com/atomix/atomix-go-framework/pkg/atomix/errors"
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
			vlan, err := lookupSwitchVlan(sw, p.Vlans.Untagged)
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

// get a unique sid for the switch from atomix
func (s *Synchronizer) getUniqueSid(ctx context.Context, switchName string) (uint32, error) {
	log.Infof("Looking for switch %s", switchName)
	entry, err := s.sidMap.Get(ctx, switchName)
	if entry != nil && err == nil {
		sid := bytesToUint32(entry.Value)
		log.Warnf("Switch found with SID %d", sid)
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

		_, err = s.sidMap.Put(ctx, switchName, sidValue)
	}
	return uint32(newSid), err
}

func (s *Synchronizer) handleSwitch(ctx context.Context, scope *FabricScope) error {
	var err error

	sw := scope.Switch

	log.Infof("Fabric %s handling switch %s", *scope.FabricId, *sw.SwitchId)

	if sw.Management == nil || sw.Management.Address == nil || sw.Management.PortNumber == nil {
		return fmt.Errorf("Fabric %s switch %s has no management address", *scope.FabricId, *sw.SwitchId)
	}

	device := &onosDevice{}

	device.Basic.Name = *sw.DisplayName
	driver := sw.Attribute["driver"]
	if driver == nil || driver.Value == nil || *driver.Value == "" {
		return errors.New("switch driver attribute must be specified")
	}

	device.Basic.Driver = *driver.Value
	device.SegmentRouting.Ipv4NodeSid, err = s.getUniqueSid(ctx, *sw.SwitchId)
	if err != nil {
		return fmt.Errorf("fabric %s switch %s unable to create SID: %s", *scope.FabricId, *sw.SwitchId, err)
	}
	device.SegmentRouting.IsEdgeRouter = sw.Role != RoleSpine // TODO: smbaker: verify with charles
	device.SegmentRouting.Ipv4Loopback = *sw.Management.Address
	device.SegmentRouting.RouterMac, err = addressToMac(*sw.Management.Address)

	pipeconf := sw.Attribute["pipeconf"]
	if pipeconf == nil || pipeconf.Value == nil || *pipeconf.Value == "" {
		return errors.New("switch pipeconf attribute must be specified")
	}
	device.Basic.PipeConf = *pipeconf.Value
	device.Basic.ManagementAddress = fmt.Sprintf("grpc://%s:%d?device_id=1", *sw.Management.Address, *sw.Management.PortNumber)
	// omit for now: locType, gridX, gridY

	// segmentRouting
	// Ipv4 Node Sid, Ipv4 Loopback, Router Mac, Is Edge Router, Adjacency Sids
	device.SegmentRouting.AdjacencySids = []uint16{}
	device.SegmentRouting.Ipv4Loopback = managementAddressToIP(*sw.Management.Address)
	//device.SegmentRouting.Ipv4NodeSid = s.getUniqueSid(device.SegmentRouting.Ipv4Loopback) // TODO: smbaker: probably of collision is not negligible
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
	err = s.pusher.PushUpdate(url, *scope.OnosUsername, *scope.OnosPassword, data)
	if err != nil {
		return 1, fmt.Errorf("Fabric %s failed to Push netconfig update: %s", *scope.FabricId, err)
	}

	s.CacheUpdate(CacheModelNetConfig, *scope.FabricId, scope.NetConfig)

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
			log.Warnf("Failed to push fabric %s: %v", fabricID, err)
		}

		pushFailuresTotal += pushFailures
		KpiSynchronizationDuration.WithLabelValues(fabricID).Observe(time.Since(tStart).Seconds())
	}

	return pushFailuresTotal, nil
}
