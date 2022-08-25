// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// GNMIPusher implements a pusher that pushes to a REST API endpoint.

package synchronizer

import (
	"context"
	"crypto/tls"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	baseClient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"math"
	"time"
)

// Client gNMI client interface
type Client interface {
	io.Closer
	Capabilities(ctx context.Context, r *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error)
	Get(ctx context.Context, r *gpb.GetRequest) (*gpb.GetResponse, error)
	Set(ctx context.Context, r *gpb.SetRequest) (*gpb.SetResponse, error)
	Subscribe(ctx context.Context, q baseClient.Query) error
	Poll() error
}

// client gnmi client
type client struct {
	client *gclient.Client
}

// Subscribe calls gNMI subscription on a given query
func (c *client) Subscribe(ctx context.Context, q baseClient.Query) error {
	err := c.client.Subscribe(ctx, q)
	go c.run(ctx)
	return errors.FromGRPC(err)
}

// Poll issues a poll request using the backing client
func (c *client) Poll() error {
	return c.client.Poll()
}

func (c *client) run(ctx context.Context) {
	log.Infof("Subscription response monitor started")
	for {
		err := c.client.Recv()
		if err != nil {
			log.Infof("Subscription response monitor stopped due to %v", err)
			return
		}
	}
}

// Capabilities returns the capabilities of the target
func (c *client) Capabilities(ctx context.Context, req *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	capResponse, err := c.client.Capabilities(ctx, req)
	return capResponse, errors.FromGRPC(err)
}

// Get calls gnmi Get RPC
func (c *client) Get(ctx context.Context, req *gpb.GetRequest) (*gpb.GetResponse, error) {
	getResponse, err := c.client.Get(ctx, req)
	return getResponse, errors.FromGRPC(err)
}

//////
// Testing hack
/////

func getClientCredentials() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}

const (
	onosConfigName = "onos-config"
	onosConfigPort = "5150"
	onosConfig     = onosConfigName + ":" + onosConfigPort
)

// GetOnosConfigDestination returns a gnmi Destination for the onos-config service
func GetOnosConfigDestination() (baseClient.Destination, error) {
	creds, err := getClientCredentials()
	if err != nil {
		return baseClient.Destination{}, err
	}

	return baseClient.Destination{
		Addrs:   []string{onosConfig},
		Target:  onosConfigName,
		TLS:     creds,
		Timeout: 10 * time.Second,
	}, nil
}

// Set calls gnmi Set RPC
func (c *client) Set(ctx context.Context, req *gpb.SetRequest) (*gpb.SetResponse, error) {
	log.Warn("client.Set()")
	dest, err := GetOnosConfigDestination()
	if err != nil {
		log.Error("Unable to get onos destination", err)
	}
	opts := []grpc.DialOption{grpc.WithBlock()}
	//opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)))
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(dest.TLS)))
	log.Warn("dialing")
	conn, err := grpc.DialContext(ctx, dest.Addrs[0], opts...)
	if err != nil {
		log.Error("Unable to dial grpc", err)
	}
	log.Warn("NewFromConn()")
	client, err := gclient.NewFromConn(ctx, conn, dest)
	if err != nil {
		log.Error("Unable to make client", err)
	}
	c.client = client
	log.Warnf("Sending set request %v", req)
	setResponse, err := c.client.Set(ctx, req)
	log.Warnf("gnmi set operation finished, result is %v", setResponse)
	return setResponse, errors.FromGRPC(err)
}

// Close closes the gnmi client
func (c *client) Close() error {
	return c.client.Close()
}
