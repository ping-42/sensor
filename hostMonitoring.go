package main

import (
	"runtime"
	"sync"
	"time"

	"github.com/ping-42/42lib/sensorTask"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/net"
)

var (
	latestHostTelemetry sensorTask.HostTelemetry
	cpuUsageLock        sync.RWMutex
)

const monitorPeriod = 5 * time.Second

func (s *Sensor) getLatestHostTelemetry() sensorTask.HostTelemetry {
	cpuUsageLock.RLock()
	defer cpuUsageLock.RUnlock()
	return latestHostTelemetry
}

// monitorHostTelementry getting CPU, RAM, NumGoroutine sample on each 5 sec
func monitorHostTelementry() {

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
		cpuUsage, err := cpu.Percent(monitorPeriod, false)
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
		netIO, err := net.IOCounters(true)
		if err != nil {
			sensorLogger.Error("Error getting net.IOCounters", err.Error())
			continue
		}
		netw := []sensorTask.Network{}
		for _, io := range netIO {
			netw = append(netw, sensorTask.Network{Name: io.Name, BytesSent: io.BytesSent, BytesRecv: io.BytesRecv})
		}

		cpuUsageLock.Lock()

		// TODO can have more than one cpu?
		latestHostTelemetry.Cpu = sensorTask.Cpu{
			ModelName: cpuInfo[0].ModelName,
			Cores:     uint16(cpuInfo[0].Cores),
			CpuUsage:  cpuUsage[0],
		}
		latestHostTelemetry.Memory = sensorTask.Memory{
			Total:       mem.Total,
			Used:        mem.Used,
			UsedPercent: mem.UsedPercent,
			Free:        mem.Free,
		}
		latestHostTelemetry.Network = netw
		latestHostTelemetry.GoRoutines = runtime.NumGoroutine()
		cpuUsageLock.Unlock()
	}
}
