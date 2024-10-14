// Ping42 Network Sensor
// This is the main file
package main

import (
	"os"
	"time"

	"github.com/ping-42/42lib/logger"
	log "github.com/sirupsen/logrus"
)

// goroutineTimeout timeout duration
const goroutineContextTimeout = 90 * time.Second

// goroutinesPoolSize adjust the limit of Goroutines
const goroutinesPoolSize = 66

// Release versioning magic
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var sensorLogger = logger.WithTestType("sensor")

func init() {
	sensorLogger.WithFields(log.Fields{
		"version":   version,
		"commit":    commit,
		"buildDate": date,
	}).Info("Starting PING42 Sensor Client")
}

func main() {

	// check if the container is privileged
	isRoot := isRoot()
	if !isRoot {
		logger.Logger.Warn("sensor is not privileged")
	} else {
		logger.Logger.Info("sensor is privileged")
	}

	// check if the container can listen for ICMP on ipv6
	ipv6Enabled := isIpv6Enabled()

	telemetryServerUrl := os.Getenv("PING42_TELEMETRY_SERVER")
	if telemetryServerUrl == "" {
		telemetryServerUrl = "wss://api.ping42.net"
	}

	cap := capabilities{
		IsRoot:      isRoot,
		Ipv6Enabled: ipv6Enabled,
	}

	// init the base sensor struct
	s := Sensor{
		telemetryServerUrl: telemetryServerUrl,
		Cap:                cap,
	}

	sensorEnvToken := os.Getenv("PING42_SENSOR_TOKEN")
	if sensorEnvToken == "" {
		sensorLogger.Error("Missing PING42_SENSOR_TOKEN environment variable!")
		os.Exit(2)
	}

	err := s.parseSensorToken(sensorEnvToken)
	if err != nil {
		sensorLogger.Error("Unable to parse PING42_SENSOR_TOKEN - please make you copied it correctly.")
		os.Exit(3)
	}

	// connect to telemetry server
	err = s.connectToTelemetryServer()
	if err != nil {
		sensorLogger.Error("Unable to establish telemetry connection")
		os.Exit(4)
	}

	// defer closing the ws connection
	defer s.WsConn.Close() // #nosec G104
	// start monitoring CPU usage, RAM... in a goroutine.
	go s.monitorHostTelementry()

	// start working
	//TODO This should probably be handled somehow in a loop with tasks being discarded upon failure
	err = s.handleTasks()
	if err != nil {
		sensorLogger.Error("error while handleTasks(): ", err.Error())
		os.Exit(5)
	}
}
