package main

import (
	"encoding/json"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/m-pavel/go-apogee/lib"
	"github.com/m-pavel/go-hassio-mqtt/pkg"
	_ "net/http"
	_ "net/http/pprof"
)

type ApogeeService struct {
	a     *apogee.Apogee
	topic string
}
type Request struct {
	Sun float32 `json:"sun"`
}

func (ts ApogeeService) PrepareCommandLineParams() {}
func (ts ApogeeService) Name() string              { return "apogee" }

func (ts *ApogeeService) Init(client MQTT.Client, topic, topicc, topica string, debug bool) error {
	var err error
	ts.a, err = apogee.FindUsbOne(&apogee.LibUsbFct{LightType: apogee.Sunlight, Debug: false})
	ts.topic = topic
	return err
}

func (ts ApogeeService) Do(client MQTT.Client) error {
	v, err := ts.a.Read()
	if err != nil {
		return err
	}

	mqt := Request{Sun: v}
	bp, err := json.Marshal(&mqt)
	if err != nil {
		return err
	}
	tkn := client.Publish(ts.topic, 0, false, bp)
	return tkn.Error()
}

func (ts ApogeeService) Close() error {
	return ts.a.Close()
}

func main() {
	hmss := ghm.NewStub(&ApogeeService{})
	hmss.Main()
}
