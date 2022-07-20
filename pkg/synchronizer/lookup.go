// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

import (
	"context"
	"fmt"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

func getTopoClient(ctx context.Context, s *Synchronizer) (topoapi.TopoClient, error) {
	opts, err := certs.HandleCertPaths(s.caPath, s.keyPath, s.certPath, true)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	opts = append(opts,
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))

	conn, err := grpc.DialContext(ctx, s.topoEndpoint, opts...)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	client := topoapi.CreateTopoClient(conn)
	return client, nil
}

func lookupFabricControllerInfo(ctx context.Context, s *Synchronizer, fabricName string) (*topoapi.ControllerInfo, error) {
	topoClient, err := getTopoClient(ctx, s)
	if err != nil {
		return nil, errors.FromGRPC(err)
	}

	getResponse, err := topoClient.Get(ctx, &topoapi.GetRequest{
		ID: topoapi.ID(fabricName),
	})
	if err != nil {
		return nil, errors.FromGRPC(err)
	}
	log.Debug("topo response object: %v", getResponse.Object)

	fabricObject := getResponse.Object
	controllerInfo := &topoapi.ControllerInfo{}
	err = fabricObject.GetAspect(controllerInfo)
	if err != nil {
		return nil, errors.FromGRPC(err)
	}
	log.Debug("controller address %v port %v", controllerInfo.ControlEndpoint.Address, controllerInfo.ControlEndpoint.Port)

	return controllerInfo, nil
}
