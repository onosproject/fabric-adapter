// SPDX-FileCopyrightText: 2022-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package synchronizer implements a synchronizer for converting sdcore gnmi to json
package synchronizer

const (
	onosRouteAppName = "org.onosproject.route-service"
)

type onosDevice struct {
	SegmentRouting struct {
		Ipv4NodeSid   uint32   `json:"ipv4NodeSid,omitempty"`
		Ipv4Loopback  string   `json:"ipv4Loopback,omitempty"`
		RouterMac     string   `json:"routerMac,omitempty"`
		IsEdgeRouter  bool     `json:"isEdgeRouter,omitempty"`
		PairDeviceID  string   `json:"pairDeviceId,omitempty"`
		PairLocalPort uint16   `json:"pairLocalPort,omitempty"`
		AdjacencySids []uint16 `json:"adjacencySids,omitempty"`
	} `json:"segmentrouting,omitempty"`
	Basic struct {
		Name              string `json:"name"`
		ManagementAddress string `json:"managementAddress,omitempty"`
		Driver            string `json:"driver"`
		PipeConf          string `json:"pipeconf"`
		LocType           string `json:"locType,omitempty"`
		GridX             uint16 `json:"gridX,omitempty"`
		GridY             uint16 `json:"gridY,omitempty"`
	} `json:"basic"`
}

type onosInterface struct {
	Ips          []string `json:"ips,omitempty"`
	VlanTagged   []uint16 `json:"vlan-tagged,omitempty"`
	VlanUntagged uint16   `json:"vlan-untagged,omitempty"`
	Name         string   `json:"name"`
}

type onosPort struct {
	Interfaces []onosInterface `json:"interfaces"`
}

type onosHost struct {
	Basic struct {
		Name      string   `json:"name"`
		Ips       []string `json:"ips"`
		Locations []string `json:"locations"`
	} `json:"basic"`
}

type onosRoute struct {
	Prefix  string `json:"prefix"`
	NextHop string `json:"nextHop"`
}

type onosUp4Config struct {
	Devices []string `json:"devices"`
}

type onosDhcpConfig struct {
	ConnectPoint string   `json:"dhcpServerConnectPoint"`
	ServerIps    []string `json:"serverIps"`
}

type onosTelemetryReport struct {
	CollectorIP               string   `json:"collectorIp"`
	CollectorPort             uint16   `json:"collectorPort"`
	MinFlowHopLatencyChangeNs uint32   `json:"minFlowHopLatencyChangeNs"`
	WatchSubnets              []string `json:"watchSubnets"`
}

type onosTelemetryQueue struct {
	TriggerNs uint32 `json:"triggerNs"`
	ResetNs   uint32 `json:"restNs"`
}

// Note: These are probably app-specific and should be
// broken out into a union of independent configs
type onosApp struct {
	Routes                                []onosRoute                   `json:"routes,omitempty"`
	Up4                                   *onosUp4Config                `json:"up4,omitempty"`
	DhcpDefault                           *onosDhcpConfig               `json:"default,omitempty"`
	TelemetryReport                       *onosTelemetryReport          `json:"report,omitempty"`
	TelemetryQueueReportLatencyThresholds map[string]onosTelemetryQueue `json:"queueReportLatencyThresholds,omitempty"`
}

// OnosNetConfig JSON Schema for an onos netcfg
type OnosNetConfig struct {
	Devices map[string]*onosDevice `json:"devices,omitempty"`
	Ports   map[string]*onosPort   `json:"ports,omitempty"`
	Hosts   map[string]*onosHost   `json:"hosts,omitempty"`
	Apps    map[string]*onosApp    `json:"apps,omitempty"`
}

// OnosComponentConfig JSON Schema for an onos component config
// NOTE: These are probably component-specific
type OnosComponentConfig struct {
	MonitorHosts bool   `json:"monitorHosts"`
	OrobeRate    string `json:"probeRate"`
	RealPortID   bool   `json:"realPortId"`
}

// OnosConfig JSON Schema for an onos config
type OnosConfig struct {
	ComponentConfig map[string]OnosComponentConfig `json:"componentConfig"`
	NetConfig       OnosNetConfig                  `json:"netcfg"`
}

// ChassisConfig JSON Schema for onos chassis config
type ChassisConfig struct {
	// TODO: populate
}
