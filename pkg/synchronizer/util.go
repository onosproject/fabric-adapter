// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// nolint deadcode
package synchronizer

import (
	"fmt"
)

// BoolToUint32 convert a boolean to an unsigned integer
func BoolToUint32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// DerefStrPtr dereference a string pointer, returning a default if it is nil
func DerefStrPtr(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// DerefUint32Ptr dereference a uint32 pointer, returning default if it is nil
func DerefUint32Ptr(u *uint32, def uint32) uint32 {
	if u == nil {
		return def
	}
	return *u
}

// DerefUint16Ptr dereference a uint16 pointer, returning default if it is nil
func DerefUint16Ptr(u *uint16, def uint16) uint16 {
	if u == nil {
		return def
	}
	return *u
}

// DerefUint8Ptr dereference a uint8 pointer, returning default if it is nil
func DerefUint8Ptr(u *uint8, def uint8) uint8 {
	if u == nil {
		return def
	}
	return *u
}

// DerefInt8Ptr dereference an int8 pointer, returning default if it is nil
func DerefInt8Ptr(u *int8, def int8) int8 {
	if u == nil {
		return def
	}
	return *u
}

// ProtoStringToProtoNumber converts a protocol name to a number
func ProtoStringToProtoNumber(s string) (uint8, error) {
	n, okay := map[string]uint8{"TCP": 6, "UDP": 17}[s]
	if !okay {
		return 0, fmt.Errorf("Unknown protocol %s", s)
	}
	return n, nil
}

// aStr facilitates easy declaring of pointers to strings
func aStr(s string) *string {
	return &s
}

// aBool facilitates easy declaring of pointers to bools
func aBool(b bool) *bool {
	return &b
}

// aInt8 facilitates easy declaring of pointers to int8
func aInt8(u int8) *int8 {
	return &u
}

// aUint8 facilitates easy declaring of pointers to uint8
func aUint8(u uint8) *uint8 {
	return &u
}

// aUint16 facilitates easy declaring of pointers to uint16
func aUint16(u uint16) *uint16 {
	return &u
}

// aUint32 facilitates easy declaring of pointers to uint32
func aUint32(u uint32) *uint32 {
	return &u
}

// aUint64 facilitates easy declaring of pointers to uint64
func aUint64(u uint64) *uint64 {
	return &u
}
