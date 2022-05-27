# SPDX-FileCopyrightText: 2022-present Intel Corporation
# SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

FROM onosproject/golang-build:v1.0 as build

ARG LOCAL_AETHER_MODELS
ARG org_label_schema_version=unknown
ARG org_label_schema_vcs_url=unknown
ARG org_label_schema_vcs_ref=unknown
ARG org_label_schema_build_date=unknown
ARG org_opencord_vcs_commit_date=unknown
ARG org_opencord_vcs_dirty=unknown

ENV ADAPTER_ROOT=$GOPATH/src/github.com/onosproject/fabric-adapter
ENV CGO_ENABLED=0

RUN mkdir -p $ADAPTER_ROOT/

COPY . $ADAPTER_ROOT/

RUN cat $ADAPTER_ROOT/go.mod

RUN cd $ADAPTER_ROOT && GO111MODULE=on go build -o /go/bin/fabric-adapter \
        -ldflags \
        "-X github.com/onosproject/fabric-adapter/internal/pkg/version.Version=$org_label_schema_version \
         -X github.com/onosproject/fabric-adapter/internal/pkg/version.GitCommit=$org_label_schema_vcs_ref  \
         -X github.com/onosproject/fabric-adapter/internal/pkg/version.GitDirty=$org_opencord_vcs_dirty \
         -X github.com/onosproject/fabric-adapter/internal/pkg/version.GoVersion=$(go version 2>&1 | sed -E  's/.*go([0-9]+\.[0-9]+\.[0-9]+).*/\1/g') \
         -X github.com/onosproject/fabric-adapter/internal/pkg/version.Os=$(go env GOHOSTOS) \
         -X github.com/onosproject/fabric-adapter/internal/pkg/version.Arch=$(go env GOHOSTARCH) \
         -X github.com/onosproject/fabric-adapter/internal/pkg/version.BuildTime=$org_label_schema_build_date" \
         ./cmd/fabric-adapter

FROM alpine:3.14
RUN apk add bash openssl curl libc6-compat

ENV HOME=/home/fabric-adapter

RUN mkdir $HOME
WORKDIR $HOME

COPY --from=build /go/bin/fabric-adapter /usr/local/bin/

