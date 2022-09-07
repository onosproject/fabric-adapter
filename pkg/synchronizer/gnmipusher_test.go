// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// RESTPusher implements a pusher that pushes to a REST API endpoint.

package synchronizer

import (
	"context"
	baseClient "github.com/openconfig/gnmi/client"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"k8s.io/kube-openapi/pkg/validation/errors"
	"net/http"
	"testing"
)

type testClient struct {
	payload        string
	expectedStatus int32
}

func (*testClient) Capabilities(ctx context.Context, r *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	return nil, nil
}
func (*testClient) Get(ctx context.Context, r *gpb.GetRequest) (*gpb.GetResponse, error) {
	return nil, nil
}
func (tc *testClient) Set(ctx context.Context, r *gpb.SetRequest) (*gpb.SetResponse, error) {
	if tc.expectedStatus == http.StatusOK {
		tc.payload = r.String()
		return nil, nil
	}
	return nil, errors.New(tc.expectedStatus, "gnmi set operation failed")
}
func (*testClient) Subscribe(ctx context.Context, q baseClient.Query) error { return nil }
func (*testClient) Poll() error                                             { return nil }
func (*testClient) Close() error                                            { return nil }

// TestGNMIPush tests that the proper payload is posted by the pusher
func TestGNMIPush(t *testing.T) {
	tc := &testClient{expectedStatus: http.StatusOK}
	pusher := NewGNMIPusherWithClient("someURL", "stratum", "somepayload", "path", tc)
	assert.NoError(t, pusher.PushUpdate())
	assert.Contains(t, tc.payload, "val:{bytes_val:\"somepayload\"")
}

// TestGNMIPusherError tests that a POST operation that the pusher properly handles an HTTP error on the POST operation
func TestGNMIPushError(t *testing.T) {
	tc := &testClient{expectedStatus: http.StatusForbidden}
	pusher := NewGNMIPusherWithClient("someURL", "stratum", "somepayload", "path", tc)
	err := pusher.PushUpdate()
	assert.Error(t, err)
	pushError := err.(*PushError)
	assert.NotNil(t, pushError)
	assert.Greater(t, pushError.StatusCode, 0)
}
