package apogee

import (
	"encoding/binary"
	"math"
	"fmt"
	"time"
	"errors"
	"log"
)

// LightType definition
type LightType byte

// LightType values
const (
	defaultPause           = 500 * time.Millisecond
	Electric     LightType = iota
	Sunlight
)

// Apogee device descriptor
type Apogee struct {
	Name string
	UUID string
	cmm  communication
	// Calibration
	offset     float32 //volts
	multiplier float32
	// Permanent calibration
	pOffset             float32 //volts
	pMultiplier         float32
	lightTypeMultiplier float32
	lightSource         LightType
	Version             int
	Serial              float32
}

type communication interface {
	Open(debug bool) error
	IsOpen() bool
	IsDebug() bool
	Read(buffer []byte) (int, error)
	Write(bytes string) (int, error)
	Close() error
}

// CmncFactory communication factory
type CmncFactory interface {
	list() ([]Apogee, error)
	register(handler HotPlugHandler)
	unregister(handler HotPlugHandler)
	Close() error
	Init()
}

func (a *Apogee) cmnc(cmd string, respsz int) ([]byte, int, error) {
	if !a.cmm.IsOpen() {
		return nil, 0, errors.New("device is not opened")
	}
	wb, err := a.cmm.Write(cmd)
	if a.cmm.IsDebug() {
		log.Printf("wrote %d bytes\n", wb)
	}
	if err != nil {
		return nil, 0, err
	}
	buffer := make([]byte, respsz)
	rb, err := a.cmm.Read(buffer)
	if err != nil {
		return nil, 0, err
	}
	if a.cmm.IsDebug() {
		log.Printf("read %d bytes\n", rb)
	}
	return buffer, rb, nil
}

// ReadRawVolts Raw readings
// Useful for calibration only
func (a *Apogee) ReadRawVolts() (float32, error) {
	buffer, _, err := a.cmnc(getVolt, 5)
	if err != nil {
		return 0, err
	}
	err = validateResponse(getVolt, buffer)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(pInt(buffer[1:])), nil
}

// Read current readings from device
// Watts/m2
func (a *Apogee) Read() (float32, error) {
	volt, err := a.ReadRawVolts()
	if err != nil {
		return volt, err
	}
	volt = (volt - a.offset) * a.multiplier * 1000.0 * a.lightTypeMultiplier
	return volt, nil
}

// SetLightSource Configure light source
func (a *Apogee) SetLightSource(t LightType) {
	a.lightSource = t
	switch t {
	case Electric:
		a.lightTypeMultiplier = 1.0
		break
	case Sunlight:
		if a.Version < 10 && a.Version >= 5 {
			a.lightTypeMultiplier = 1.12
		} else {
			a.lightTypeMultiplier = 1.14
		}
		break
	}
}

// GetLightSource get device light source
func (a *Apogee) GetLightSource() LightType {
	return a.lightSource
}

func (a *Apogee) readVersion() error {
	buffer, _, err := a.cmnc(getFirmwareVersion, 4)
	if err != nil {
		return err
	}
	err = validateResponse(getFirmwareVersion, buffer)
	if err != nil {
		return err
	}
	a.Version = int(buffer[1])
	return nil
}

func (a *Apogee) readCalibration() error {
	buffer, _, err := a.cmnc(readCalibration, 9)
	if err != nil {
		return err
	}
	err = validateResponse(readCalibration, buffer)
	if err != nil {
		return err
	}

	a.multiplier = math.Float32frombits(pInt(buffer[1:5]))
	a.offset = math.Float32frombits(pInt(buffer[5:]))
	return nil
}

func (a *Apogee) readPermanentCalibration() error {
	buffer, _, err := a.cmnc(readPermanentCalibration, 9)
	if err != nil {
		return err
	}
	err = validateResponse(readPermanentCalibration, buffer)
	if err != nil {
		return err
	}

	a.pMultiplier = math.Float32frombits(pInt(buffer[1:5]))
	a.pOffset = math.Float32frombits(pInt(buffer[5:]))
	return nil
}

func (a *Apogee) readSerial() error {

	buffer, _, err := a.cmnc(readSerialNum, 5)
	if err != nil {
		return err
	}
	err = validateResponse(readSerialNum, buffer)
	if err != nil {
		return err
	}
	a.Serial = math.Float32frombits(pInt(buffer[1:]))
	return nil
}

func pInt(b []byte) uint32 {
	if len(b) != 4 {
		panic("4 bytes array is expected")
	}
	sdata := make([]byte, 4)
	sdata[0] = b[3]
	sdata[1] = b[2]
	sdata[2] = b[1]
	sdata[3] = b[0]
	return binary.BigEndian.Uint32(sdata)
}

func validateResponse(req string, resp []byte) error {
	if resp[0] != req[0] {
		return errors.New("invalid response prefix")
	}
	return nil
}

// Close device
func (a *Apogee) Close() error {
	return a.cmm.Close()
}

func (a *Apogee) String() string {
	return fmt.Sprintf("%s [%s], FW version %d, Serial %f, Offset %f, Multiplier %f", a.Name, a.UUID, a.Version, a.Serial, a.offset, a.multiplier)
}

// GetCalibration read device calibration
func (a *Apogee) GetCalibration() (float32, float32) {
	return a.multiplier, a.offset
}

// GetPermanentCalibration read permanent device calibration
func (a *Apogee) GetPermanentCalibration() (float32, float32) {
	a.readPermanentCalibration()
	return a.pMultiplier, a.pOffset
}

// SetCalibration save device calibration
func (a *Apogee) SetCalibration(multiplier, offset float32) error {
	ms := pString(math.Float32bits(multiplier))
	os := pString(math.Float32bits(offset))
	req := fmt.Sprintf(setCalibration, ms, os)
	buffer, _, err := a.cmnc(req, 9)
	if err != nil {
		return err
	}
	err = validateResponse(setCalibration, buffer)
	if err != nil {
		return err
	}

	a.multiplier = math.Float32frombits(pInt(buffer[1:5]))
	a.offset = math.Float32frombits(pInt(buffer[5:]))
	return nil
}

// GetSensorType provide device type
func (a *Apogee) GetSensorType() (string, error) {
	buffer, rb, err := a.cmnc(getSensorType, 9)
	if err != nil {
		return "", err
	}
	err = validateResponse(getSensorType, buffer)
	if err != nil {
		return "", err
	}

	return string(buffer[1 : rb-1]), nil
}

func pString(v uint32) string {
	var res, o []byte
	res = make([]byte, 4)
	o = make([]byte, 4)
	binary.BigEndian.PutUint32(res, v)
	i := len(res)
	for _, c := range res {
		i--
		o[i] = c
	}
	return string(o)
}
