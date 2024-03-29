// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// GNMIPusher implements a pusher that pushes to a REST API endpoint.

package synchronizer

import (
	"context"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"time"
)

// GnmiPushClientFactory is used to create the underlying GNMI clients. Overridden by tests
var GnmiPushClientFactory = newClient

// GNMIPusher implements a pusher that pushes to a gnmi endpoint.
type GNMIPusher struct {
	endpoint   string
	path       string
	payload    string
	target     string
	pushClient Client
}

const (
	// SecureConnection : use a certificate secured connection
	SecureConnection = true

	// InsecureConnection : use a plain text connection
	InsecureConnection = false
)

func newClient(dest string, target string, secure bool) Client {
	gpc := &client{
		dest:   dest,
		secure: secure,
		target: target,
	}
	return gpc
}

// NewGNMIPusher allocates a gnmi pusher for a given endpoint
func NewGNMIPusher(url string, target string, payload string, path string, secureConnection bool) PusherInterface {
	gpc := GnmiPushClientFactory(url, target, secureConnection)
	return NewGNMIPusherWithClient(url, target, payload, path, gpc)
}

// NewGNMIPusherWithClient allocates a gnmi pusher for a given endpoint
func NewGNMIPusherWithClient(url string, target string, payload string, path string, pushClient Client) PusherInterface {
	gnmiPusher := &GNMIPusher{
		endpoint:   url,
		pushClient: pushClient,
		payload:    payload,
		target:     target,
		path:       path,
	}

	return gnmiPusher
}

// PushUpdate pushes an update to the GNMI server.
func (p *GNMIPusher) PushUpdate() error {
	setGnmiRequest := &gnmiapi.SetRequest{}

	var es []*gnmiapi.PathElem
	if p.path != "" {
		e := &gnmiapi.PathElem{
			Name: p.path,
		}
		es = []*gnmiapi.PathElem{e}
	}
	path := &gnmiapi.Path{
		Origin: "",
		Elem:   es,
		Target: p.target,
	}
	tv := &gnmiapi.TypedValue{
		Value: &gnmiapi.TypedValue_BytesVal{
			BytesVal: []byte(p.payload),
		},
	}
	ud := &gnmiapi.Update{
		Path:       path,
		Val:        tv,
		Duplicates: 0,
	}
	uds := []*gnmiapi.Update{ud}

	setGnmiRequest.Replace = uds

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := p.pushClient.Set(ctx, setGnmiRequest)
	if err != nil {
		return &PushError{
			Endpoint:   p.endpoint,
			StatusCode: 500, // Not sure what the right thing to do is
			Status:     err.Error(),
			Operation:  "SET",
		}
	}
	return nil
}

// PushDelete pushes a delete operation to the GNMI server
func (p *GNMIPusher) PushDelete() error {
	return nil
}
