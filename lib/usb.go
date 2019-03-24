package apogee

import (
	"errors"
)

const (
	getVolt                  = "U!"
	readCalibration          = "\x83!"
	setCalibration           = "\x84%s%s!"
	getLoggingInterval       = "\xf1!"
	getLoggingCount          = "\xf3!"
	getLoggedEntry           = "\xf2%s!"
	readSerialNum            = "\x87!"
	eraseLoggedData          = "\xf4!"
	setLoggingInterval       = "\xf0%s%s%s%s!"
	readPermanentCalibration = "\x85!"
	overwritePermanent       = "\x86%s%s!"
	setSerialNum             = "\x88%s!"
	getFirmwareVersion       = "\xf5!"
	getSensorType            = "\xf6!"
)

const (
	apogee = 0x1916
	sp420  = 0x0031
)

// HotPlugHandler hotplug apogee wrapper
type HotPlugHandler interface {
	Added(apogee Apogee)
	Removed(apogee Apogee)
}

// FindUsbOne - Return first attached device
// Or error if no device found
// Note that device is opened and must be closed with Close() method after use
func FindUsbOne(factory CmncFactory) (*Apogee, error) {
	devices, err := FindUsb(factory)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, errors.New("no device found")
	}
	return &devices[0], err
}

// FindUsb - Return list of attached Apogee devices
// Note that each device must be closed with Close() method after use
func FindUsb(factory CmncFactory) ([]Apogee, error) {
	return factory.list()
}

// RegisterHandler register hotplug handler
func RegisterHandler(factory CmncFactory, handler HotPlugHandler) {
	factory.register(handler)
}

// UnRegisterHandler unregister hotplug handler
func UnRegisterHandler(factory CmncFactory, handler HotPlugHandler) {
	factory.unregister(handler)
}
