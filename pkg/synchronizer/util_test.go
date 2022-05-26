// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements the synchronizer.
package synchronizer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProtoStringToProtoNumber(t *testing.T) {
	n, err := ProtoStringToProtoNumber("UDP")
	assert.Nil(t, err)
	assert.Equal(t, uint8(17), n)

	n, err = ProtoStringToProtoNumber("TCP")
	assert.Nil(t, err)
	assert.Equal(t, uint8(6), n)

	_, err = ProtoStringToProtoNumber("MQTT")
	assert.EqualError(t, err, "Unknown protocol MQTT")
}
