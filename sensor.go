package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/ping-42/42lib/logger"
	"github.com/ping-42/42lib/sensor"
	"github.com/ping-42/42lib/wss"
	log "github.com/sirupsen/logrus"

	"encoding/base64"
	"encoding/json"

	"io"

	gohttp "net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ping-42/42lib/dns"
	"github.com/ping-42/42lib/http"
	"github.com/ping-42/42lib/icmp"
)

// Sensor main sensor struct
type Sensor struct {
	sensorId     uuid.UUID
	sensorSecret string

	WsConn net.Conn
	Tasks  map[sensor.TaskName]sensor.TaskRunner
}

// Build a JWT Token and connect to the telemetry server
func (s *Sensor) connectToTelemetryServer() (err error) {

	telemetryServerUrl := os.Getenv("PING42_TELEMETRY_SERVER")
	if telemetryServerUrl == "" {
		telemetryServerUrl = "wss://api.ping42.net"
	}

	// Retry logic
	for attempt := 1; attempt <= 3000; attempt++ {

		jwtToken, er := s.buildJwtToken()
		if er != nil {
			sensorLogger.WithFields(log.Fields{
				"error": er.Error(),
			}).Error("Unable to build a JWT Token.")
			return
		}

		wsConn, err := wsConnectToServer(telemetryServerUrl, jwtToken)
		if err != nil {
			sensorLogger.WithFields(log.Fields{
				"telemetryServer": telemetryServerUrl,
				"error":           fmt.Errorf("%v", err),
				"attempt":         attempt,
			}).Error("Unable to connect to telemetry server")

			// Exponential backoff for retries
			backoff := time.Duration(10*attempt) * time.Second

			time.Sleep(backoff)
			continue
		}

		sensorLogger.WithFields(log.Fields{
			"localAddr":       wsConn.LocalAddr(),
			"remoteAddr":      wsConn.RemoteAddr().String(),
			"telemetryServer": telemetryServerUrl,
		}).Info("Connection to telemetry server established...")

		// store the connection
		s.WsConn = wsConn
		return nil
	}

	return fmt.Errorf("unable to connect to telemetry server after multiple attempts")
}

func (s *Sensor) handleTasks() (err error) {

	// goroutines worker pool - limit the active goroutines
	pool := make(chan struct{}, goroutinesPoolSize)

	for {
		msg, op, er := wsutil.ReadServerData(s.WsConn)
		if er != nil {
			sensorLogger.WithFields(log.Fields{
				"error": er.Error(),
			}).Error("Failed to read task message from server!")

			if er == io.EOF {
				sensorLogger.Error("Connection lost. Attempting to reconnect...")
				if err := s.connectToTelemetryServer(); err != nil {
					return err
				}
				continue
			}
			return
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
		return
	}

	sensorLogger.Info("task response sent to server")

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
		icmpTask, er := http.NewTaskFromBytes(msg)
		if er != nil {
			err = er
			return
		}
		resultTask = icmpTask

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
// func wsConnectToServer(telemetryServerUrl string, jwtToken string) (conn net.Conn, err error) {
// 	dialURL := fmt.Sprintf("%v/?sensor_token=%v", telemetryServerUrl, url.QueryEscape(jwtToken))
// 	conn, _, _, err = ws.Dial(context.Background(), dialURL)
// 	return
// }

func wsConnectToServer(telemetryServerUrl string, jwtToken string) (conn net.Conn, err error) {
	//dialURL := fmt.Sprintf("%v/?sensor_token=%v", telemetryServerUrl, url.QueryEscape(jwtToken))

	header := ws.HandshakeHeaderHTTP(gohttp.Header{
		"Authorization": []string{jwtToken},
	})

	dialer := ws.Dialer{
		Header: header,
	}
	conn, _, _, err = dialer.Dial(context.Background(), telemetryServerUrl)

	return
}
