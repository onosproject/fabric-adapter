// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// RESTPusher implements a pusher that pushes to a REST API endpoint.

package synchronizer

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	restPusherTestPayload = "REST POST payload"
)

// TestRestPush tests that the proper payload is posted by the pusher
func TestRestPush(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, string(body), restPusherTestPayload)
	}))
	defer ts.Close()

	b := []byte(restPusherTestPayload)
	pusher := NewRestPusher(ts.URL, "u", "p", b)
	assert.NoError(t, pusher.PushUpdate())
}

// TestRestPusherError tests that a POST operation that the pusher properly handles an HTTP error on the POST operation
func TestRestPushError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	b := []byte(restPusherTestPayload)
	pusher := NewRestPusher(ts.URL, "u", "p", b)
	err := pusher.PushUpdate()
	assert.Error(t, err)
	pushError := err.(*PushError)
	assert.NotNil(t, pushError)
	assert.Equal(t, http.StatusForbidden, pushError.StatusCode)
}
