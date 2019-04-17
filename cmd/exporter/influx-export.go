package main

import (
	"flag"
	"os"
	"syscall"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/sevlyar/go-daemon"

	"net/http"
	"log"
	"github.com/m-pavel/go-apogee/lib"

)

func main() {
	var logf = flag.String("log", "exporter.log", "log")
	var pid = flag.String("pid", "exporter.pid", "pid")
	var notdaemonize = flag.Bool("n", false, "Do not do to background.")
	var signal = flag.String("s", "", `send signal to the daemon stop — shutdown`)
	var iserver = flag.String("influx", "http://localhost:8086", "http://localhost:8086")
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lshortfile)

	daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGTERM, termHandler)

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
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	daemonf(*iserver)
}

func daemonf(server string) {

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: server,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "sunlight",
		Precision: "ms",
	})
	if err != nil {
		log.Fatal(err)
	}
	cmnc := apogee.LibUsbFct{LightType: apogee.Sunlight, Debug: false}
	device, err := apogee.FindUsbOne(&cmnc)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()
	failcnt := 0
	for {
		exit := false
		select {
		case <-stop:
			exit = true
			break
		case <-time.After(time.Second):
		}

		if failcnt >= 10 {
			break
		}

		if exit {
			break
		}

		v, err := readData(device)
		if err != nil {
			log.Printf("%d - %v\n",err,failcnt)
			failcnt+=1
		} else {
			err = logData(v, c, bp)
			if err != nil {
				log.Printf("%d - %v\n",err,failcnt)
				failcnt+=1
			} else {
				log.Printf("Written %f\n", v)
				failcnt = 0
			}
		}
	}
	done <- struct{}{}
}

func readData(device *apogee.Apogee) (float32, error) {
	light, err := device.Read()
	if err != nil {
		return 0, err
	}
	return light, nil
}

func logData(v float32, c client.Client, bp client.BatchPoints) error {
	// Create a point and add to batch
	tags := map[string]string{"light": "light-average"}
	fields := map[string]interface{}{
		"value": v,
	}

	pt, err := client.NewPoint("light", tags, fields, time.Now())
	if err != nil {
		return err
	}
	bp.AddPoint(pt)

	err = c.Write(bp)
	return err
}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func termHandler(sig os.Signal) error {
	log.Println("terminating...")
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}
