package main

import (
	"log"

	"github.com/zllovesuki/ROGManager/system/atkacpi"
)

func main() {
	ctrlSet, err := atkacpi.NewAtkControl(atkacpi.WriteControlCode)
	if err != nil {
		panic(err)
	}

	inputBuf := make([]byte, atkacpi.KeyPressControlBufferLength)
	copy(inputBuf, atkacpi.KeyPressControlBuffer)
	inputBuf[atkacpi.KeyPressControlByteIndex] = 197 // emulate pressing the ROG Key

	writeOutput, err := ctrlSet.Write(inputBuf)
	if err != nil {
		panic(err)
	}
	log.Println(writeOutput)

	ctrlGet, err := atkacpi.NewAtkControl(atkacpi.ReadControlCode)
	if err != nil {
		panic(err)
	}

	readOutput, err := ctrlGet.Read(atkacpi.KeyPressControlOutputBufferLength)
	if err != nil {
		panic(err)
	}
	log.Println(readOutput)
}
