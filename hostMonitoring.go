package main

import (
	"context"
	"runtime"
	"strings"

	"github.com/ping-42/42lib/constants"
	"github.com/ping-42/42lib/sensor"
	"github.com/ping-42/42lib/wss"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/v3/cpu"
	gopsutilnet "github.com/shirou/gopsutil/v3/net"
)

// monitorHostTelementry getting CPU, RAM, NumGoroutine sample on each 5 sec
func (s *Sensor) monitorHostTelementry() {
	ctx := context.Background() //TODO think for timeouts
	for {
		// CPU info
		cpuInfo, err := cpu.Info()
		if err != nil {
			sensorLogger.Error("Error getting CPU info", err.Error())
			continue
		}
		if len(cpuInfo) == 0 {
			sensorLogger.Error("Error getting CPU info", "expect len(cpuInfo)>0")
			continue
		}

		// CPU usage
		cpuUsage, err := cpu.Percent(constants.TelemetryMonitorPeriod, false)
		if err != nil {
			sensorLogger.Error("Error getting cpuUsage", err.Error())
			continue
		}
		if len(cpuUsage) == 0 {
			sensorLogger.Error("Error getting CPU usage", "expect len(cpuUsage)>0")
			continue
		}

		// Mem usage
		mem, err := mem.VirtualMemory()
		if err != nil {
			sensorLogger.Error("Error getting VirtualMemory", err.Error())
			continue
		}

		// Network usage
		netIO, err := gopsutilnet.IOCounters(true)
		if err != nil {
			sensorLogger.Error("Error getting net.IOCounters", err.Error())
			continue
		}
		netw := []sensor.Network{}
		for _, io := range netIO {
			netw = append(netw, sensor.Network{
				Name:        io.Name,
				BytesSent:   io.BytesSent,
				BytesRecv:   io.BytesRecv,
				PacketsSent: io.PacketsSent,
				PacketsRecv: io.PacketsRecv,
			})
		}

		var hostTelemetry sensor.HostTelemetry
		hostTelemetry.MessageGeneralType = wss.MessageGeneralType(wss.MessageTypeTelemtry)
		// TODO can have more than one cpu?
		hostTelemetry.Cpu = sensor.Cpu{
			ModelName: cpuInfo[0].ModelName,
			//#nosec G115
			Cores:    uint16(cpuInfo[0].Cores),
			CpuUsage: cpuUsage[0],
		}
		hostTelemetry.Memory = sensor.Memory{
			Total:       mem.Total,
			Used:        mem.Used,
			UsedPercent: mem.UsedPercent,
			Free:        mem.Free,
		}
		hostTelemetry.Network = netw
		hostTelemetry.GoRoutines = runtime.NumGoroutine()

		err = hostTelemetry.SendToServer(ctx, s.WsConn)
		if err != nil {
			sensorLogger.Error("hostTelemetry.SendToServer, ", err.Error())

			if strings.Contains(err.Error(), "connection lost") ||
				strings.Contains(err.Error(), "broken pipe") {
				sensorLogger.Error("Attempting to reconnect to the telemetry server from monitorHostTelementry...")

				// Attempt reconnection
				reconnectErr := s.reconnect()
				if reconnectErr != nil {
					sensorLogger.Error("Reconnection failed: ", reconnectErr.Error())
					return
				}

				sensorLogger.Info("Reconnected to telemetry server")
			}

			continue
		}
	}
}
