package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/ping-42/42lib/logger"
	"github.com/ping-42/42lib/sensorTask"
	log "github.com/sirupsen/logrus"

	"encoding/base64"
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/ping-42/42lib/dns"
	"github.com/ping-42/42lib/http"
	"github.com/ping-42/42lib/icmp"
)

// Sensor main sensor struct
type Sensor struct {
	sensorId     string
	sensorSecret string

	WsConn net.Conn
	Tasks  map[sensorTask.TaskName]sensorTask.TaskRunner
}

func (s *Sensor) connectToWsServer() (err error) {

	jwtToken, err := s.buildJwtToken()
	if err != nil {
		sensorLogger.Error("buildAuthToken() error", err.Error())
		return
	}

	wsConn, err := wsConnectToServer(jwtToken)
	if err != nil {
		sensorLogger.Error("wsConnectToServer() error", err.Error())
		return
	}
	s.WsConn = wsConn
	return
}

func (s *Sensor) handleTasks() (err error) {

	// goroutines worker pool - limit the active goroutines
	pool := make(chan struct{}, goroutinesPoolSize)

	for {
		msg, op, er := wsutil.ReadServerData(s.WsConn)
		if er != nil {
			sensorLogger.Error("failed to read message from server", er.Error())
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
	var response = sensorTask.TResult{
		TaskId:   task.GetId(),
		TaskName: task.GetName(),
	}

	// the actual task execution
	res, err := task.Run(ctx)
	if err != nil {
		logger.LogError(err.Error(), "error in task.Run()", sensorLogger)
		response.Error = fmt.Errorf("error in task.Run(), %v", err).Error()
	}
	response.Result = res

	// get and set the latest host telemetry as cpu, mem..
	response.HostTelemetry = s.getLatestHostTelemetry()

	err = response.SendToServer(ctx, s.WsConn)
	if err != nil {
		// this error will not be sent to the server, will need some mechanism for sending/pulling to the server
		logger.LogError(err.Error(), "error in SendToServer()", sensorLogger)
		return
	}

	sensorLogger.Info("task response sent to server")

}

func (s *Sensor) factoryTask(ctx context.Context, msg []byte) (resultTask sensorTask.TaskRunner, err error) {

	// check if the context is done
	if ctx.Err() != nil {
		err = fmt.Errorf("context done detected in factoryTask:%v", ctx.Err())
		return
	}

	baseTask := sensorTask.Task{}
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
	if s.sensorId == "" || s.sensorSecret == "" {
		err = fmt.Errorf("missing sensorSecret or sensorId")
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

func (s *Sensor) parseEnvToken() (err error) {

	sensorEnvToken := os.Getenv("PING42_SENSOR_TOKEN")
	if sensorEnvToken == "" {
		err = fmt.Errorf("Missing PING42_SENSOR_TOKEN in the env var!")
		return
	}

	t, err := base64.StdEncoding.DecodeString(sensorEnvToken)
	if err != nil {
		err = fmt.Errorf("base64.StdEncoding.DecodeString sensorEnvToken: %v", err)
		return
	}

	parsed := strings.Split(string(t), ".")
	if len(parsed) != 2 {
		err = fmt.Errorf("unexpected token struct")
		return
	}

	s.sensorId = parsed[0]
	s.sensorSecret = parsed[1]
	return
}

func wsConnectToServer(jwtToken string) (conn net.Conn, err error) {

	// TODO get the URL from the ENVs
	// TODO send token as header
	dialURL := fmt.Sprintf("ws://localhost:8080/?sensor_token=%v", url.QueryEscape(jwtToken))

	conn, _, _, err = ws.Dial(context.Background(), dialURL)

	if err != nil {
		err = fmt.Errorf("cannot connect:%v", err)
		return
	}
	sensorLogger.WithFields(log.Fields{
		"LocalAddr": conn.LocalAddr(),
	}).Info("Connected to server")
	return
}
