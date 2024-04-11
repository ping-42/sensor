// Ping42 Network Sensor
// This is the main file
package main

import (
	"fmt"
	"time"

	"github.com/ping-42/42lib/logger"
	// telemetry "github.com/ping-42/sensor/src/utils"
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
	sensorLogger.Info(fmt.Sprintf("Sensor Starting - version %s (commit %s) built %s", version, commit, date))
}

func main() {

	// init the base sensor struct
	s := Sensor{}

	err := s.parseEnvToken()
	if err != nil {
		sensorLogger.Error("parseEnvToken err!")
		return
	}

	// connect to ws server
	err = s.connectToWsServer()
	if err != nil {
		sensorLogger.Error("error while connectToWsServer()", err.Error())
		return
	}

	// close the ws connection
	defer s.WsConn.Close()

	// start monitoring CPU usage, RAM... in a goroutine.
	go monitorHostTelementry()

	// start working
	err = s.handleTasks()
	if err != nil {
		sensorLogger.Error("error while handleTasks()", err.Error())
		return
	}
}
