package main

import (
	"fmt"
	"time"

	mc "github.com/zllovesuki/G14Manager/cxx/MatrixController"
)

func main() {

	fmt.Println("initializing controller")
	controller, err := mc.NewController()
	if err != nil {
		panic(err)
	}

	fmt.Println("clearing LEDs")
	if err := controller.Clear(); err != nil {
		panic(err)
	}

	half := []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f}
	buf := make([]byte, 1815, 1815)
	copy(buf, half)
	for j := len(half); j < len(buf); j *= 2 {
		copy(buf[j:], buf[:j])
	}

	fmt.Println("turning on all LEDs to half brightness")
	if err := controller.Draw(buf); err != nil {
		panic(err)
	}

	fmt.Println("wait 5 seconds")
	time.Sleep(time.Second * 5)

	fmt.Println("clearing LEDs")
	if err := controller.Clear(); err != nil {
		panic(err)
	}

}
