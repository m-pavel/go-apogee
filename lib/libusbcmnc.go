package apogee

import (
	"time"
	"errors"
	"strings"
	"fmt"
	"runtime"
	"github.com/nkovacs/gousb"

	"log"
)

// LibUsbFct libUsb implementation
type LibUsbFct struct {
	h         []HotPlugHandler
	d         []Apogee
	hpctx     *gousb.Context
	LightType LightType
	Debug     bool
}

// libUsbCmnc- libusb communication implementation
type libUsbCmnc struct {
	vid    gousb.ID
	pid    gousb.ID
	ctx    *gousb.Context
	dev    *gousb.Device
	intf   *gousb.Interface
	conf   *gousb.Config
	output *gousb.OutEndpoint
	input  *gousb.InEndpoint
	done   func()
	debug  bool
}

func (c *libUsbCmnc) IsDebug() bool {
	return c.debug
}

func (c *libUsbCmnc) IsOpen() bool {
	return c.output != nil && c.input != nil
}

func (c *libUsbCmnc) Read(buffer []byte) (i int, e error) {
	i, e = c.input.Read(buffer)
	if c.debug {
		if e != nil {
			log.Printf("read %v", e)
		} else {
			log.Printf("read %x\n", []byte(buffer))
		}

	}
	return
}

func (c *libUsbCmnc) Write(bytes string) (i int, e error) {
	i, e = c.output.Write([]byte(bytes))
	if c.debug {
		if e != nil {
			log.Printf("write %v", e)
		} else {
			log.Printf("write %x\n", []byte(bytes))
		}
	}
	return
}

func (c *libUsbCmnc) Close() error {
	var err error
	if c.intf != nil {
		c.intf.Close()
		c.intf = nil
	}
	if c.conf != nil {
		c.conf.Close()
		c.conf = nil
	}
	if c.dev != nil {
		c.dev.Close()
		c.dev = nil
	}
	if c.done != nil {
		c.done()
		c.done = nil
	}
	if c.ctx != nil {
		err = c.ctx.Close()
		c.ctx = nil
	}

	return err
}

func (c *libUsbCmnc) Open(debug bool) error {
	c.debug = debug
	c.ctx = gousb.NewContext()
	var err error
	c.dev, err = c.ctx.OpenDeviceWithVIDPID(c.vid, c.pid)
	if err != nil {
		log.Println(err)
		c.Close()
		return err
	}
	if c.dev == nil {
		return errors.New("no device, no error")
	}
	c.dev.SetAutoDetach(true)
	if debug {
		log.Printf(c.dev.ConfigDescription(1)) // Only 1
	}
	c.conf, err = c.dev.Config(1)
	if err != nil {
		log.Println(err)
		c.Close()
		return err
	}
	c.intf, err = c.conf.Interface(1, 0)
	if err != nil {
		log.Println(err)
		c.Close()
		return err
	}
	//c.intf, c.done, err = c.dev.DefaultInterface()
	if debug {
		log.Println(c.intf)
	}
	if err != nil {
		log.Println(err)
		c.Close()
		return err
	}
	if debug {
		log.Println(c.intf.Setting)
	}
	c.output, err = c.intf.OutEndpoint(2)
	if err != nil {
		log.Println(err)
		c.Close()
		return err
	}
	c.output.Timeout = 3 * time.Second
	if debug {
		log.Println(c.output)
	}

	c.input, err = c.intf.InEndpoint(2)
	if err != nil {
		log.Println(err)
		c.Close()
		return err
	}
	c.input.Timeout = 3 * time.Second
	if debug {
		log.Println(c.input)
	}
	return nil
}

func (l *LibUsbFct) get(desc *gousb.DeviceDesc) communication {
	device := libUsbCmnc{vid: desc.Vendor, pid: desc.Product}
	return &device
}

func (l *LibUsbFct) list() ([]Apogee, error) {
	l.Init()
	return l.d, nil
}

// Init factory
func (l *LibUsbFct) Init() {
	if l.hpctx == nil {
		l.hpctx = gousb.NewContext()
		l.platformInit()
	}
}

func (l *LibUsbFct) mkApogee(d *gousb.DeviceDesc) (*Apogee, error) {
	ap := Apogee{Name: "SP-420", UUID: deviceUUID(d)}
	ap.cmm = l.get(d)
	ap.SetLightSource(l.LightType)
	err := ap.cmm.Open(l.Debug)
	if err != nil {
		log.Printf("Error opening %s : %s\n", d, err)
		if strings.Contains(err.Error(), "device or resource busy [code -6]") {
			log.Println("Try to rmmod cdc_acm.")
		}
		if strings.Contains(err.Error(), "not supported [code -12]") && runtime.GOOS == "windows" {
			log.Println("Try to uninstall Apogee drivers.")
		}
		ap.Close()
		return nil, err
	}

	initWRetry := func(fnc func(a *Apogee) error, entity string) error {
		if l.Debug {
			log.Printf("Reading %s\n", entity)
		}
		err = fnc(&ap)
		if err != nil {
			time.Sleep(defaultPause)
			if l.Debug {
				log.Printf("Retrying reading %s\n", entity)
			}
			err = fnc(&ap)
			if err != nil {
				log.Printf("Error getting %s with retry of the device %s : %s\n", entity, d, err)
			}
			return err
		}
		return nil
	}

	initWRetry((*Apogee).readCalibration, "calibration")
	time.Sleep(defaultPause)
	initWRetry((*Apogee).readVersion, "version")
	time.Sleep(defaultPause)
	initWRetry((*Apogee).readSerial, "serial")
	time.Sleep(defaultPause)
	initWRetry((*Apogee).readCalibration, "calibration")

	return &ap, nil
}

// Close libusb
func (l *LibUsbFct) Close() error {
	var err error
	if l.hpctx != nil {
		err = l.hpctx.Close()
		l.hpctx = nil
	}
	return err
}

func deviceUUID(d *gousb.DeviceDesc) string {
	return fmt.Sprintf("%d:%d", d.Bus, d.Address)
}

func isSupported(dd *gousb.DeviceDesc) bool {
	return dd.Vendor == gousb.ID(apogee) && dd.Product == gousb.ID(sp420)
}
