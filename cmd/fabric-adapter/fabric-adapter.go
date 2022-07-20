// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/gnxi/utils/credentials"
	"github.com/onosproject/fabric-adapter/internal/pkg/version"
	synchronizer "github.com/onosproject/fabric-adapter/pkg/synchronizer"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/sdcore-adapter/pkg/diagapi"
	"github.com/onosproject/sdcore-adapter/pkg/gnmi"
	"github.com/onosproject/sdcore-adapter/pkg/target"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	bindAddr             = flag.String("bind_address", ":10161", "Bind to address:port or just :port")
	metricAddr           = flag.String("metric_address", ":9851", "Prometheus metric endpoint bind to address:port or just :port")
	partialUpdateDisable = flag.Bool("partial_update_disable", false, "Disable partial update; send full updates to core on every change")
	postDisable          = flag.Bool("post_disable", false, "Disable posting to connectivity service endpoints")
	postTimeout          = flag.Duration("post_timeout", time.Second*10, "Timeout duration when making post requests")
	aetherConfigAddr     = flag.String("aether_config_addr", "", "If specified, pull initial state from aether-config at this address")
	aetherConfigTarget   = flag.String("aether_config_target", "connectivity-service-v4", "Target to use when pulling from aether-config")
	showModelList        = flag.Bool("show_models", false, "Show list of available modes")
	diagsPort            = flag.Uint("diags_port", 8080, "Port to use for Diagnostics API")
	caPath               = flag.String("caPath", "", "path to CA certificate")
	keyPath              = flag.String("keyPath", "", "path to client private key")
	certPath             = flag.String("certPath", "", "path to client certificate")
	topoEndpoint         = flag.String("topoEndpoint", "onos-topo:5150", "onos-topo endpoint address")
)

var log = logging.GetLogger("fabric-adapter")

func serveMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(*metricAddr, nil); err != nil {
		log.Fatalf("failed to serve metrics: %v", err)
	}
}

// Synchronize and eat the error. This lets aether-config know we applied the
// configuration, but leaves us to retry applying it to the southbound device
// ourselves.
func synchronizerWrapper(s synchronizer.SynchronizerInterface) gnmi.ConfigCallback {
	return func(config *gnmi.ConfigForest, callbackType gnmi.ConfigCallbackType, target string, path *pb.Path) error {
		err := s.Synchronize(config, callbackType, target, path)
		if err != nil {
			// Report the error, but do not send the error upstream.
			log.Warnf("Error during synchronize: %v", err)
		}
		return nil
	}
}

func main() {
	var sync synchronizer.SynchronizerInterface

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	log.Infof("fabric-adapter")
	version.LogVersion("  ")

	// Initialize the synchronizer's service-specific code.
	log.Infof("Initializing synchronizer")
	sync = synchronizer.NewSynchronizer(
		synchronizer.WithPostEnable(!*postDisable),
		synchronizer.WithPartialUpdateEnable(!*partialUpdateDisable),
		synchronizer.WithPostTimeout(*postTimeout),
		synchronizer.WithCertPaths(*caPath, *keyPath, *certPath),
		synchronizer.WithTopoEndpoint(*topoEndpoint),
	)

	// The synchronizer will convey its list of models.
	model := sync.GetModels()

	if *showModelList {
		fmt.Fprintf(os.Stdout, "Supported models:\n")
		for _, m := range model.SupportedModels() {
			fmt.Fprintf(os.Stdout, "  %s\n", m)
		}
		return
	}

	opts := credentials.ServerCredentials()
	g := grpc.NewServer(opts...)

	c := make(chan os.Signal, 1)
	signal.Notify(c)

	s, err := target.NewTarget(model, synchronizerWrapper(sync))
	if err != nil {
		log.Fatalf("error in creating gnmi target: %v", err)
	}

	sync.Start()

	go func() {
		for {
			oscall := <-c
			if oscall.String() == "terminated" || oscall.String() == "interrupt" {
				log.Warnf("system call:%+v", oscall)
				s.Close()
				os.Exit(0)
			}
		}
	}()

	pb.RegisterGNMIServer(g, s)
	reflection.Register(g)

	log.Info("starting metric handler")
	go serveMetrics()

	log.Infof("starting out-of-band API on %d", *diagsPort)
	diagapi.StartDiagnosticAPI(s, *aetherConfigAddr, *aetherConfigTarget, *diagsPort)

	log.Infof("starting to listen on %s", *bindAddr)
	listen, err := net.Listen("tcp", *bindAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Info("starting to serve")
	if err := g.Serve(listen); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
