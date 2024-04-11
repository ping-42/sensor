package telemetry

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/ping-42/42lib/icmp"
)

// SendPingStatsToServer sends the icmp ping stats to the server
func SendPingStatsToServer(data map[string]icmp.IcmpPingResult, conn net.Conn) (err error) {
	telemetryData, err := json.Marshal(data)
	if err != nil {
		log.Println("Cannot serialize data: ", err)
		return err
	}
	// send data to server
	err = wsutil.WriteClientMessage(conn, ws.OpText, telemetryData)
	if err != nil {
		log.Println("Cannot send: ", err)

	}
	log.Println("Telemetry data sent to server:  " + string(telemetryData))

	// read response from server
	serverResponse, _, err := wsutil.ReadServerData(conn)
	if err != nil {
		log.Println("Cannot receive data: ", err)

	}
	log.Println("Response from server from Server: ", string(serverResponse))

	// wait 5 seconds
	time.Sleep(time.Duration(5) * time.Second)
	// close connection
	return nil
}
