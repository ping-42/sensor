package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/ping-42/42lib/logger"
	"github.com/ping-42/42lib/sensor"
	"github.com/ping-42/42lib/traceroute"
	"github.com/ping-42/42lib/wss"
	log "github.com/sirupsen/logrus"

	"encoding/base64"
	"encoding/json"

	gohttp "net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ping-42/42lib/dns"
	"github.com/ping-42/42lib/http"
	"github.com/ping-42/42lib/icmp"
	netICMP "golang.org/x/net/icmp"
)

// Sensor main sensor struct
type Sensor struct {
	sensorId     uuid.UUID
	sensorSecret string
	Cap          capabilities

	WsConn net.Conn
	Tasks  map[sensor.TaskName]sensor.TaskRunner

	mu            sync.Mutex
	reconnectOnce sync.Once
	reconnectDone chan error

	telemetryServerUrl string
}

// capabilities holds the sensor's capabilities and can be extended to include more fields
type capabilities struct {
	IsRoot      bool
	Ipv6Enabled bool
}

// isRoot checks if the proccess is running with root privileges
// effective user id root=0
func isRoot() bool {
	return os.Geteuid() == 0
}

// isIpv6Enabled checks if the sensor can listen for ICMP on ipv6
func isIpv6Enabled() bool {
	ipv6Conn, err := netICMP.ListenPacket("ip6:ipv6-icmp", "::")
	if err != nil {
		sensorLogger.Warn("sensor cannot listen on ipv6: ", err)
		return false
	}
	sensorLogger.Info("sensor can listen on ipv6: ", ipv6Conn)
	return true
}

// ensure that only one reconnect operation runs at a time
func (s *Sensor) reconnect() error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reconnectDone == nil {
		s.reconnectDone = make(chan error)
	}

	// sync.Once to ensure the reconnection logic is executed only once
	s.reconnectOnce.Do(func() {
		go func() {
			if s.WsConn != nil {
				s.WsConn.Close() // #nosec G104
			}

			err := s.connectToTelemetryServer()

			s.reconnectDone <- err
			close(s.reconnectDone)
		}()
	})

	// block until reconnection is completed (waiting for the reconnectDone channel to be closed)
	err := <-s.reconnectDone

	// reset sync.Once so future reconnections can run if needed
	s.reconnectOnce = sync.Once{}
	s.reconnectDone = nil

	return err
}

func (s *Sensor) connectToTelemetryServer() (err error) {
	// retry logic
	for attempt := 1; attempt <= 30000; attempt++ {

		// build JWT token
		jwtToken, er := s.buildJwtToken()
		if er != nil {
			sensorLogger.WithFields(log.Fields{
				"error": er.Error(),
			}).Error("Unable to build a JWT Token.")
			return er
		}

		// attempt WebSocket connection
		wsConn, err := wsConnectToServer(s.telemetryServerUrl, jwtToken)
		if err != nil {
			// Log error with retry attempt
			sensorLogger.WithFields(log.Fields{
				"telemetryServer": s.telemetryServerUrl,
				"error":           fmt.Errorf("%v", err),
				"attempt":         attempt,
			}).Error("Unable to connect to telemetry server")

			// exponential backoff for retries with a cap (e.g., max 10min)
			backoff := time.Duration(10*attempt) * time.Second
			if backoff > 10*time.Minute {
				backoff = 10 * time.Minute
			}

			// adding jitter to backoff to avoid synchronized retries
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond // #nosec G404
			time.Sleep(backoff + jitter)

			continue
		}

		// log successful connection
		sensorLogger.WithFields(log.Fields{
			"localAddr":       wsConn.LocalAddr(),
			"remoteAddr":      wsConn.RemoteAddr().String(),
			"telemetryServer": s.telemetryServerUrl,
		}).Info("Connection to telemetry server established...")

		// store the WebSocket connection
		s.WsConn = wsConn
		return nil
	}

	return fmt.Errorf("unable to connect to telemetry server after multiple attempts")
}

func (s *Sensor) handleTasks() (err error) {

	// goroutines worker pool - limit the active goroutines
	pool := make(chan struct{}, goroutinesPoolSize)

	for {

		// ensure WebSocket connection is established before handling tasks
		if s.WsConn == nil {
			sensorLogger.Error("WebSocket connection is not established. Attempting to reconnect...")
			if err := s.reconnect(); err != nil {
				return fmt.Errorf("failed to connect to telemetry server: %v", err)
			}
			// continue the loop after reconnecting
			continue
		}

		msg, op, er := wsutil.ReadServerData(s.WsConn)
		if er != nil {
			sensorLogger.WithFields(log.Fields{
				"error": er.Error(),
			}).Error("Failed to read task message from server!")

			sensorLogger.Error("Connection issue. Attempting to reconnect...")
			if err := s.reconnect(); err != nil {
				return fmt.Errorf("failed to reconnect: %w", err)
			}
			// continue the loop after reconnecting
			continue
		}

		if op != ws.OpText {
			continue
		}

		// acquire a worker slot
		pool <- struct{}{}

		// create a context with a timeout (e.g., 60 seconds)
		ctx, cancel := context.WithTimeout(context.Background(), goroutineContextTimeout)
		defer cancel()

		// run the sensor task
		// e.g. dsn, icmp...
		go s.doTask(ctx, pool, msg)
	}
}

func (s *Sensor) doTask(ctx context.Context, pool <-chan struct{}, msg []byte) {

	defer func() {
		// release the worker slot when done
		<-pool
	}()

	// based on the msg, choose which task needs to be executed
	task, err := s.factoryTask(ctx, msg)

	sensorLogger.Info("received task:", string(msg))
	if err != nil {
		// this error will not be sent to the server, will need some mechanism for sending/pulling to the server
		logger.LogError(err.Error(), "error in factoryTask()", sensorLogger)
		return
	}

	// init the logger
	sensorLogger := sensorLogger.WithFields(log.Fields{
		"task_name": task.GetName(),
		"task_id":   task.GetId(),
	})

	// init the task reponse that will be sent to the server
	var response = sensor.TResult{
		MessageGeneralType: wss.MessageGeneralType(wss.MessageTypeTaskResult),
		TaskId:             task.GetId(),
		TaskName:           task.GetName(),
	}

	// the actual task execution
	res, err := task.Run(ctx)
	if err != nil {
		logger.LogError(err.Error(), "error in task.Run()", sensorLogger)
		response.Error = fmt.Errorf("error in task.Run(), %v", err).Error()
	}
	response.Result = res

	err = response.SendToServer(ctx, s.WsConn)
	if err != nil {
		// this error will not be sent to the server, will need some mechanism for sending/pulling to the server
		logger.LogError(err.Error(), "error in SendToServer()", sensorLogger)

		// basic socket errors should be ignored and reconnections should happen
		if strings.Contains(err.Error(), "connection lost") ||
			strings.Contains(err.Error(), "broken pipe") {
			sensorLogger.Error("Attempting to reconnect to the telemetry server from doTask...")

			// attempt reconnection to the telemetry server
			// if this fails, we will panic for now and perhaps try to pool up the data locally
			// until we can establish a connection again
			reconnectErr := s.reconnect()
			if reconnectErr != nil {
				sensorLogger.Error("Reconnection failed: ", reconnectErr.Error())
				panic("Unable to reconnect to server")
			}

			sensorLogger.Info("Reconnected to telemetry server, attempting to resend data")

			// second try to send the data
			err = response.SendToServer(ctx, s.WsConn)
			if err != nil {
				logger.LogError(err.Error(), "error in SendToServer() after reconnecting", sensorLogger)
				return
			}
			sensorLogger.Info("Task response sent to server after reconnect")
			return
		}
		return
	}

	sensorLogger.Info("Task response sent to server")

}

func (s *Sensor) factoryTask(ctx context.Context, msg []byte) (resultTask sensor.TaskRunner, err error) {

	// check if the context is done
	if ctx.Err() != nil {
		err = fmt.Errorf("context done detected in factoryTask:%v", ctx.Err())
		return
	}

	baseTask := sensor.Task{}
	err = json.Unmarshal(msg, &baseTask)
	if err != nil {
		err = fmt.Errorf("can not build base Task from the received task:%v, %v", string(msg), err)
		return
	}

	if baseTask.Id == uuid.Nil {
		err = fmt.Errorf("can not build base Task with nil ID: %v", string(msg))
		return
	}

	switch baseTask.Name {
	case dns.TaskName:
		dnsTask, er := dns.NewTaskFromBytes(msg)
		if er != nil {
			err = er
			return
		}
		resultTask = dnsTask

	case icmp.TaskName:
		icmpTask, er := icmp.NewTaskFromBytes(msg)
		if er != nil {
			err = er
			return
		}
		resultTask = icmpTask

	case http.TaskName:
		httpTask, er := http.NewTaskFromBytes(msg)
		if er != nil {
			err = er
			return
		}
		resultTask = httpTask

	case traceroute.TaskName:
		tracerouteTask, er := traceroute.NewTaskFromBytes(msg)
		if er != nil {
			err = er
			return
		}
		resultTask = tracerouteTask

	default:
		err = fmt.Errorf("unexpected Task Name:%v, %v", baseTask.Name, string(msg))
		return
	}

	return resultTask, nil
}

func (s *Sensor) buildJwtToken() (jwtToken string, err error) {
	if s.sensorSecret == "" {
		err = fmt.Errorf("missing sensorSecret")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sensorId": s.sensorId,
			"exp":      time.Now().Add(time.Second * 40).Unix(),
		})

	jwtToken, err = token.SignedString([]byte(s.sensorSecret))
	if err != nil {
		err = fmt.Errorf("token.SignedString: %v", err)
		return
	}

	return
}

// Parses a Sensor Token for use in authentication and telemetry submission
func (s *Sensor) parseSensorToken(sensorEnvToken string) (err error) {

	// Parses a Sensor Token to creds
	t, err := base64.StdEncoding.DecodeString(sensorEnvToken)
	if err != nil {
		err = fmt.Errorf("base64.StdEncoding.DecodeString sensorEnvToken: %v", err)
		return
	}

	var sensorCreds sensor.Creds
	err = json.Unmarshal(t, &sensorCreds)
	if err != nil {
		err = fmt.Errorf("can not unmershal sensorEnvToken: %v, %v", err, string(t))
		return
	}

	if sensorCreds.Secret == "" {
		err = fmt.Errorf("empty Secret in sensorEnvToken: %v", string(t))
		return
	}

	s.sensorId = sensorCreds.SensorId
	s.sensorSecret = sensorCreds.Secret
	return
}

// Establish a Websocket connection to the telemetry server
func wsConnectToServer(telemetryServerUrl string, jwtToken string) (conn net.Conn, err error) {
	header := ws.HandshakeHeaderHTTP(gohttp.Header{
		"Authorization": []string{jwtToken},
		"SensorVersion": []string{version},
	})

	dialer := ws.Dialer{
		Header: header,
	}
	conn, _, _, err = dialer.Dial(context.Background(), telemetryServerUrl)

	return
}
