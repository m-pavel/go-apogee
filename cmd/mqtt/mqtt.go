package main

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/m-pavel/go-apogee/lib"
	"github.com/m-pavel/go-hassio-mqtt/pkg"
	_ "net/http"
	_ "net/http/pprof"
)

type ApogeeService struct {
	a *apogee.Apogee
}
type Request struct {
	Sun float32 `json:"sun"`
}

func (ts ApogeeService) PrepareCommandLineParams() {}
func (ts ApogeeService) Name() string              { return "apogee" }

func (ts *ApogeeService) Init(client MQTT.Client, topic, topicc, topica string, debug bool, ss ghm.SendState) error {
	var err error
	ts.a, err = apogee.FindUsbOne(&apogee.LibUsbFct{LightType: apogee.Sunlight, Debug: false})
	return err
}

func (ts ApogeeService) Do() (interface{}, error) {
	v, err := ts.a.Read()
	if err != nil {
		return nil, err
	}

	return &Request{Sun: v}, nil
}

func (ts ApogeeService) Close() error {
	return ts.a.Close()
}

func main() {
	ghm.NewStub(&ApogeeService{}).Main()
}
