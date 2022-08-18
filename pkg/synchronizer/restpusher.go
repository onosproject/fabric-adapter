// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// RESTPusher implements a pusher that pushes to a REST API endpoint.

package synchronizer

import (
	"bytes"
	"net/http"
	"time"
)

// RESTPusher implements a pusher that pushes to a rest endpoint.
type RESTPusher struct {
	endpoint string
	username string
	password string
	data     []byte
}

// NewRestPusher allocates a rest pusher for a given endpoint
func NewRestPusher(url string, username string, password string, data []byte) PusherInterface {
	restPusher := &RESTPusher{
		endpoint: url,
		username: username,
		password: password,
		data:     data,
	}

	return restPusher
}

// PushUpdate pushes an update to the REST endpoint.
func (p *RESTPusher) PushUpdate() error {

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	log.Infof("Push Update endpoint=%s data=%s", p.endpoint, string(p.data))
	reader := bytes.NewReader(p.data)
	req, err := http.NewRequest(http.MethodPost, p.endpoint, reader)
	if err != nil {
		return err
	}

	req.SetBasicAuth(p.username, p.password)
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	resp, err := client.Do(req)

	/* In the future, PUT will be the correct operation
	resp, err := httpPut(client, endpoint, "application/json", data)
	*/

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Infof("Put returned status %s", resp.Status)

	if (resp.StatusCode < 200) || (resp.StatusCode >= 300) {
		return &PushError{Operation: "POST", Endpoint: p.endpoint, StatusCode: resp.StatusCode, Status: resp.Status}
	}

	return nil
}

// PushDelete pushes a delete to the REST endpoint
func (p *RESTPusher) PushDelete() error {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	log.Infof("Push Delete endpoint=%s", p.endpoint)

	req, err := http.NewRequest("DELETE", p.endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Infof("Delete returned status %s", resp.Status)

	if (resp.StatusCode < 200) || (resp.StatusCode >= 300) {
		return &PushError{Operation: "DELETE", Endpoint: p.endpoint, StatusCode: resp.StatusCode, Status: resp.Status}
	}

	return nil
}
