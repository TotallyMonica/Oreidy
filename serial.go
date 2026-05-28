package main

import (
	"fmt"

	"go.bug.st/serial"
)

var serialPort serial.Port

func writeToPort(data []byte) error {
	if serialPort == nil {
		return fmt.Errorf("port has not been opened yet")
	}
	if _, err := serialPort.Write(data); err != nil {
		return fmt.Errorf("error writing to port: %w", err)
	}
	return nil
}

func switchToPort(portNumber int) error {
	if portNumber < 1 || portNumber > 4 {
		return fmt.Errorf("invalid HDMI port %d", portNumber)
	}

	return writeToPort([]byte(fmt.Sprintf("PA%dR", portNumber)))
}

func openPort(portName string) error {
	var err error
	serialPort, err = serial.Open(portName, &serial.Mode{
		BaudRate: 9600,
	})
	if err != nil {
		return err
	}
	return nil
}

func closePort() error {
	if serialPort == nil {
		return nil
	}
	return serialPort.Close()
}
