package main

import (
	"flag"
	"log"
	_ "net/http"
	_ "net/http/pprof"
	"os"
	"syscall"
	"time"

	"net/http"

	"encoding/json"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/m-pavel/go-apogee/lib"
	"github.com/sevlyar/go-daemon"
)

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func main() {
	var logf = flag.String("log", "mqttapogee.log", "log")
	var pid = flag.String("pid", "mqttapogee.pid", "pid")
	var notdaemonize = flag.Bool("n", false, "Do not do to background.")
	var signal = flag.String("s", "", `send signal to the daemon stop — shutdown`)
	var mqtt = flag.String("mqtt", "tcp://localhost:1883", "MQTT endpoint")
	var topic = flag.String("t", "nn/apogee", "MQTT topic")
	var user = flag.String("mqtt-user", "", "MQTT user")
	var pass = flag.String("mqtt-pass", "", "MQTT password")

	var interval = flag.Int("interval", 30, "Interval secons")
	flag.Parse()
	daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGTERM, termHandler)
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)

	cntxt := &daemon.Context{
		PidFileName: *pid,
		PidFilePerm: 0644,
		LogFileName: *logf,
		LogFilePerm: 0640,
		WorkDir:     "/tmp",
		Umask:       027,
		Args:        os.Args,
	}

	if !*notdaemonize && len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Fatalf("Unable send signal to the daemon: %v", err)
		}
		daemon.SendCommands(d)
		return
	}

	if !*notdaemonize {
		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatal(err)
		}
		if d != nil {
			return
		}
	}

	daemonf(*mqtt, *topic, *user, *pass, *interval)

}

func daemonf(mqtt, topic string, u, p string, interval int) {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	opts := MQTT.NewClientOptions().AddBroker(mqtt)
	opts.SetClientID("temper-go-cli")
	if u != "" {
		opts.Username = u
		opts.Password = p
	}

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	cmnc := apogee.LibUsbFct{LightType: apogee.Sunlight, Debug: false}
	device, err := apogee.FindUsbOne(&cmnc)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()

	erinr := 0
	for {
		select {
		case <-stop:
			log.Println("Exiting")
			break
		case <-time.After(time.Duration(interval) * time.Second):
			v, err := device.Read()
			if err == nil {
				req := Request{Sun: v}
				bp, err := json.Marshal(&req)
				if err != nil {
					log.Println(err)
				}
				tkn := client.Publish(topic, 0, false, bp)
				if tkn.Error() != nil {
					log.Println(tkn.Error())
				}
			} else {
				erinr++
			}
			if erinr == 10 {
				return
			}
		}
	}

	done <- struct{}{}
}

type Request struct {
	Sun float32 `json:"sun"`
}

func termHandler(sig os.Signal) error {
	log.Println("Terminating...")
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}
