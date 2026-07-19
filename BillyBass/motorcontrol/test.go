package main

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

var (
	head1 gpio.PinIO
	head2 gpio.PinIO
)

func mustGetPin(name string) gpio.PinIO {
	pin := gpioreg.ByName(name)
	if pin == nil {
		log.Fatalf("GPIO pin %s not found", name)
	}
	return pin
}

func stopHead() {
	head1.Out(gpio.Low)
	head2.Out(gpio.Low)
	fmt.Println("HEAD: stopped")
}

func moveHeadOut(durationMs int) {
	fmt.Println("HEAD: stopping before direction change")
	stopHead()
	time.Sleep(50 * time.Millisecond)

	fmt.Println("HEAD: moving out (head1=HIGH, head2=LOW)")
	if err := head1.Out(gpio.High); err != nil {
		log.Fatal(err)
	}
	if err := head2.Out(gpio.Low); err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Duration(durationMs) * time.Millisecond)
	//stopHead()
}

func moveHeadIn(durationMs int) {
	fmt.Println("HEAD: stopping before direction change")
	stopHead()
	time.Sleep(50 * time.Millisecond)

	fmt.Println("HEAD: moving in (head1=LOW, head2=HIGH)")
	if err := head1.Out(gpio.Low); err != nil {
		log.Fatal(err)
	}
	if err := head2.Out(gpio.High); err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Duration(durationMs) * time.Millisecond)
	stopHead()
}

func main() {
	_, err := host.Init()
	if err != nil {
		log.Fatal(err)
	}

	head1 = mustGetPin("17")
	head2 = mustGetPin("22")

	// make sure we start clean
	stopHead()
	time.Sleep(500 * time.Millisecond)

	// tweak these durations (ms) until movement looks right
	outDuration := 800
	inDuration  := 800

	fmt.Println("=== Moving head OUT ===")
	moveHeadOut(outDuration)
	time.Sleep(10000 * time.Millisecond)

	fmt.Println("=== Moving head IN ===")
	moveHeadIn(inDuration)
	time.Sleep(1000 * time.Millisecond)

	fmt.Println("=== Done ===")
}
