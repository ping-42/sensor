package main

import (
	"context"
	"testing"
)

func TestInitDns(t *testing.T) {

	s := Sensor{}

	msg := []byte(`{"Id":"687914cf-67c1-4e7e-a699-108fa2574931","Name":"DNS_TASK","DnsOpts":{"Host":"https://google.com","Proto":"udp"}}`)

	// based on the msg, choose which test needs to be executed
	test, err := s.factoryTask(context.Background(), msg)
	if err != nil {
		t.Errorf("s.FactoryTask err:%v", err.Error())
		return
	}

	res, err := test.Run(context.Background())
	if err != nil {
		t.Errorf("task.Run err:%v", err.Error())
		return
	}

	_ = res

	//// Check the type of the response
	//_, isExpectedType := res.(sensorTask.TResult) // Replace YourExpectedType with the actual expected type.
	//
	//if !isExpectedType {
	//	t.Errorf("Result type is not as expected. Got %T, expected %T", res, YourExpectedType)
	//}
}

// TODO: please see why the test is faling
// func TestInitIcmp(t *testing.T) {
// 	s := Sensor{}
// 	s.prepareTasks()

// 	msg := []byte(`{"Id":"123","Name":"ICMP_TASK","DnsOpts":{"Host":"https://google.com","Proto":"udp"}}`)

// 	// based on the msg, choose which test needs to be executed
// 	test, err := s.factoryTask(context.Background(), msg)
// 	if err != nil {
// 		t.Errorf("s.FactoryTask err:%v", err.Error())
// 		return
// 	}

// 	res, err := test.Run(context.Background(), msg)
// 	if err != nil {
// 		t.Errorf("task.Run err:%v", err.Error())
// 		return
// 	}

// 	_ = res

// 	//// Check the type of the response
// 	//_, isExpectedType := res.(sensorTask.TResult) // Replace YourExpectedType with the actual expected type.
// 	//
// 	//if !isExpectedType {
// 	//	t.Errorf("Result type is not as expected. Got %T, expected %T", res, YourExpectedType)
// 	//}
// }
