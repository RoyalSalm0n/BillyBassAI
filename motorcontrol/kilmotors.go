package main

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"log"
	"fmt"
	"periph.io/x/host/v3"
)


var (
	head1  gpio.PinIO
	head2  gpio.PinIO
	mouth1 gpio.PinIO
	mouth2 gpio.PinIO
	tail1  gpio.PinIO
	tail2  gpio.PinIO
)


func mustGetPin(name string) gpio.PinIO {
        pin := gpioreg.ByName(name)
        if pin == nil {
                log.Fatalf("GPIO pin %s not found", name)
        }
        return pin
}

func stopMotors() {
        if err := mouth1.Out(gpio.Low); err != nil {
                fmt.Println("Cant find mouth1")
        }
        if err := mouth2.Out(gpio.Low); err != nil {
                fmt.Println("Cant find mouth2")
        }
        if err := head1.Out(gpio.Low); err != nil {
                fmt.Println("Cant find head1")
        }
        if err := head2.Out(gpio.Low); err != nil {
                fmt.Println("Cant find head2")
        }
        if err := tail1.Out(gpio.Low); err != nil {
                fmt.Println("Cant find tail1")
        }
        if err := tail2.Out(gpio.Low); err != nil {
                fmt.Println("Cant find tail2")
        }
}

func main() {
     if _, err := host.Init(); err != nil {
        log.Fatal(err)
    }
    head1 = mustGetPin("17")
    head2 = mustGetPin("22")
    mouth1 = mustGetPin("23")
    mouth2 = mustGetPin("24")
    tail1 = mustGetPin("5")
    tail2 = mustGetPin("6")
    stopMotors()
    defer stopMotors()



}
