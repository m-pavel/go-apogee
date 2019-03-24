// +build !windows

package apogee

import (

	"github.com/nkovacs/gousb"
	"log"
)

// work via hotplug
func (l *LibUsbFct) platformInit() {
	l.hpctx.RegisterHotplug(l.usbevt)
}

func (l *LibUsbFct) usbevt(e gousb.HotplugEvent) {
	if l.Debug {
		log.Printf("Got event %v\n", e)
	}
	dd, err := e.DeviceDesc()
	if err != nil {
		log.Println(err)
		return
	}

	if isSupported(dd) {
		var ap *Apogee
		switch e.Type() {
		case gousb.HotplugEventDeviceArrived:
			ap, err = l.mkApogee(dd)
			if err != nil {
				log.Println(err)
				return
			}
			l.d = append(l.d, *ap)
			break
		case gousb.HotplugEventDeviceLeft:
			for i := range l.d {
				if deviceUUID(dd) == l.d[i].UUID {
					ap = &l.d[i]
					l.d = append(l.d[:i], l.d[i+1:]...)
					break
				}
			}
			break
		}

		for i := range l.h {
			switch e.Type() {
			case gousb.HotplugEventDeviceArrived:
				l.h[i].Added(*ap)
				break
			case gousb.HotplugEventDeviceLeft:
				l.h[i].Removed(*ap)
				break
			}
		}
	}
}

func (l *LibUsbFct) register(handler HotPlugHandler) {
	l.h = append(l.h, handler)
}

func (l *LibUsbFct) unregister(handler HotPlugHandler) {
	for i := range l.h {
		if handler == l.h[i] {
			l.h = append(l.h[:i], l.h[i+1:]...)
			return
		}
	}
}
