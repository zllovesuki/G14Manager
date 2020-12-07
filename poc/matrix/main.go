package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
	"unicode"

	mc "github.com/zllovesuki/G14Manager/cxx/MatrixController"
)

func fillBuffer(in []byte, out []byte) {
	var i int
	for _, ch := range in {
		if !unicode.IsSpace(rune(ch)) {
			switch {
			case string(ch) == "o":
				out[i] = 0x7f
			default:
				out[i] = 0x00
			}
			i++
		}
	}
}

func main() {

	buf := make([]byte, 1815, 1815)
	line := make([]byte, 33, 33)
	scanner := bufio.NewScanner(os.Stdin)
	offset := 0
	for scanner.Scan() {
		fillBuffer(scanner.Bytes(), line)
		copy(buf[offset:], line)
		offset += 33
	}

	fmt.Println("initializing controller")
	controller, err := mc.NewController()
	if err != nil {
		panic(err)
	}

	fmt.Println("clearing LEDs")
	if err := controller.Clear(); err != nil {
		panic(err)
	}

	fmt.Println("drawing buffer")
	if err := controller.Draw(buf); err != nil {
		panic(err)
	}

	fmt.Println("wait 5 seconds")
	time.Sleep(time.Second * 5)

	fmt.Println("clearing LEDs")
	if err := controller.Clear(); err != nil {
		panic(err)
	}

	fmt.Println("freeing controller")
	controller.Close()

}
