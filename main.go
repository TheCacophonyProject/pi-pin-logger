// Used for logging the pin status on a Raspberry Pi. First designed for logging
// a PIR sensor.

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

type pin struct {
	Name string
	Gpio string
	Pull string
	Edge string
}

type configs struct {
	Pins []pin
}

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err)
	}
}

func waitForPin(done chan string, pin gpio.PinIO, name string) {
	for {
		pin.WaitForEdge(-1)
		done <- name
	}
}

func runMain() error {

	pullMap := map[string]gpio.Pull{
		"Float":    gpio.Float,
		"Down":     gpio.PullDown,
		"Up":       gpio.PullUp,
		"NoChange": gpio.PullNoChange,
	}

	edgeMap := map[string]gpio.Edge{
		"None":    gpio.NoEdge,
		"Rising":  gpio.RisingEdge,
		"Falling": gpio.FallingEdge,
		"Both":    gpio.BothEdges,
	}

	// Load config and log files
	var configFilename string
	var loggingFilename string
	if len(os.Args) == 3 {
		configFilename = os.Args[1]
		loggingFilename = os.Args[2]
	} else {
		return fmt.Errorf("requires two arguments, config file and logging file")
	}
	f, err := os.OpenFile(loggingFilename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0755)
	w := io.MultiWriter(os.Stdout, f)
	if err != nil {
		return fmt.Errorf("could not open loging file")
	}
	defer f.Close()
	log.SetOutput(w)
	buf, err := ioutil.ReadFile(configFilename)
	if err != nil {
		return err
	}
	source := []byte(buf)
	var config configs
	err = yaml.Unmarshal(source, &config)
	if err != nil {
		log.Fatalf("error: %+v", err)
	}
	log.Printf("Config: %+v", config)

	log.Println("host initialisation")
	if _, err := host.Init(); err != nil {
		return err
	}

	pins := make([]gpio.PinIO, len(config.Pins))

	done := make(chan string)
	for i, pinConfig := range config.Pins {
		edge, edgeFound := edgeMap[pinConfig.Edge]
		if !edgeFound {
			return fmt.Errorf("did not find pin edge type: %v", pinConfig.Edge)
		}
		pull, pullFound := pullMap[pinConfig.Pull]
		if !pullFound {
			return fmt.Errorf("did not find pin pull type: %v", pinConfig.Pull)
		}
		pins[i] = gpioreg.ByName(pinConfig.Gpio)
		pins[i].In(pull, edge)
		go waitForPin(done, pins[i], pinConfig.Name)
	}

	for {
		triggeredPinName := <-done
		log.Printf("Pin triggered: %v", triggeredPinName)
		for i := 0; i < len(config.Pins); i++ {
			log.Printf("\t%v:\t%v\n", config.Pins[i].Name, pins[i].Read())
		}
	}
}
