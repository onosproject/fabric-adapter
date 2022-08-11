#!/bin/sh
# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

# remove the old code and regenerate,
# if we don't remove the old code it is possible that removed protos are left behind
find ./go/ -name "*.pb.go" -type f -delete

proto_path="./stratum:${GOPATH}/src/github.com/gogo/protobuf/protobuf:${GOPATH}/src/github.com/gogo/protobuf:${GOPATH}/src:/go/src/github.com/gogo/protobuf:${GOPATH}/src/github.com/p4lang/p4runtime/proto"

### Go Protobuf code generation
go_import_paths="Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types"
go_import_paths="${go_import_paths},Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types"
go_import_paths="${go_import_paths},Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types"
go_import_paths="${go_import_paths},Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types"
go_import_paths="${go_import_paths},Monos/config/device/types.proto=github.com/onosproject/onos-api/go/onos/config/device"
go_import_paths="${go_import_paths},Monos/config/admin/admin.proto=github.com/onosproject/onos-api/go/onos/config/admin"
go_import_paths="${go_import_paths},Monos/ransim/types/types.proto=github.com/onosproject/onos-api/go/onos/ransim/types"
go_import_paths="${go_import_paths},Monos/config/v2/object.proto=github.com/onosproject/onos-api/go/onos/config/v2"
go_import_paths="${go_import_paths},Monos/config/v2/failure.proto=github.com/onosproject/onos-api/go/onos/config/v2"
go_import_paths="${go_import_paths},Monos/config/v2/value.proto=github.com/onosproject/onos-api/go/onos/config/v2"
go_import_paths="${go_import_paths},Monos/config/v2/transaction.proto=github.com/onosproject/onos-api/go/onos/config/v2"
go_import_paths="${go_import_paths},Monos/config/v2/proposal.proto=github.com/onosproject/onos-api/go/onos/config/v2"
go_import_paths="${go_import_paths},Monos/config/v2/configuration.proto=github.com/onosproject/onos-api/go/onos/config/v2"


# stratum common
protoc --proto_path=$proto_path \
    --gogofaster_out=$go_import_paths,import_path=stratum/hal/lib/common/,plugins=grpc:./go \
    stratum/stratum/hal/lib/common/*.proto

