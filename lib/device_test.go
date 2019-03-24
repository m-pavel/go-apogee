package apogee

import (
	"fmt"
	"testing"
	"time"
)

func _TestReadLibUsb(t *testing.T) {
	cmnc := LibUsbFct{Debug: false, LightType: Sunlight}
	device, err := FindUsbOne(&cmnc)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	fmt.Println(device)
	fmt.Printf("%s\n", device.String())
	for {
		//for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 500)
		fl, err := device.Read()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(fl)
	}
}

func _TestMem(t *testing.T) {

	cmnc := LibUsbFct{Debug: false, LightType: Sunlight}
	device, err := FindUsbOne(&cmnc)
	if err != nil {
		t.Fatal(err)
	}
	device.Close()

}

type testHandler struct{}

func (testHandler) Added(apogee Apogee) {
	fmt.Printf("Added %v\n", apogee)
}
func (testHandler) Removed(apogee Apogee) {
	fmt.Printf("Removed %v\n", apogee)
}

func _TestFake(t *testing.T) {

	cmnc := FakeCmncFct{}
	//cmnc := LibUsbFct{Debug: true, LightType: Sunlight}
	cmnc.Init()

	device, err := FindUsbOne(&cmnc)
	if err != nil {
		fmt.Println(err)
	} else {
		f, err := device.Read()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(f)
		fmt.Println(device)
		defer device.Close()
	}

	h := &testHandler{}
	cmnc.register(h)

	time.Sleep(time.Second * 30)
	cmnc.unregister(h)

}
