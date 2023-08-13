package main

import (
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"os"
	"time"
)

const (
	baudRate       = 9600
	readTimeout    = time.Second
	dataSize       = 8
	parity         = serial.ParityNone
	stopBits       = serial.Stop1
	defaultPort    = "/dev/ttyUSB0"
	minFMFrequency = 87.50
	maxFMFrequency = 108.00
)

var (
	getBand        = []byte{1, 20, 43, 2, 192}
	getBlend       = []byte{1, 20, 49, 2, 186}
	getDevice      = []byte{1, 20, 20, 2, 215}
	getFMFrequency = []byte{1, 20, 45, 2, 190}
	getAMFrequency = []byte{1, 20, 44, 2, 191}
	getFMMute      = []byte{1, 20, 47, 2, 188}
	getPower       = []byte{1, 20, 21, 2, 214}
	setFMBand      = []byte{1, 22, 129, 2, 104}
	turnOffBlend   = []byte{1, 21, 49, 0, 2, 185}
	turnOffMute    = []byte{1, 21, 47, 0, 2, 187}
	turnOffPower   = []byte{1, 21, 21, 0, 2, 213}
	turnOnBlend    = []byte{1, 21, 49, 1, 2, 184}
	turnOnMute     = []byte{1, 21, 47, 1, 2, 186}
	turnOnPower    = []byte{1, 21, 21, 1, 2, 212}
)

func main() {
	fmt.Println("nad-tuner")

	var (
		fmfrequencyPtr = flag.Float64("fm", 0, "Specify the frequency (e.g., 96.80)")
		showPtr        = flag.Bool("show", false, "Show power state, band, and frequency")
		powerPtr       = flag.String("power", "", "Turn power on or off")
		blendPtr       = flag.String("blend", "", "Turn blend on or off")
		mutePtr        = flag.String("mute", "", "Turn mute on or off")
		portPtr        = flag.String("port", defaultPort, "Serial port name")
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if !(*showPtr || *powerPtr != "" || *blendPtr != "" || *mutePtr != "" || *fmfrequencyPtr > 0) {
		flag.Usage()
		return
	}

	serialPort, err := openSerialPort(*portPtr)
	if err != nil {
		log.Fatal(err)
	}
	defer serialPort.Close()

	handleCommands(serialPort, *powerPtr, *blendPtr, *mutePtr, *showPtr, *fmfrequencyPtr)
}

func handleCommands(serialPort *serial.Port, argPower string, argBlend string, argMute string, argShow bool, argFrequency float64) {
	if argPower == "on" {
		power(serialPort, true)
	}

	if argPower == "off" {
		power(serialPort, false)
	}

	if argBlend == "on" {
		blend(serialPort, true)
	}

	if argBlend == "off" {
		blend(serialPort, false)
	}
	if argMute == "on" {
		mute(serialPort, true)
	}

	if argMute == "off" {
		mute(serialPort, false)
	}

	if argFrequency > 0 {
		setFMFrequency(serialPort, argFrequency)
	}

	if argShow || argFrequency > 0 || argBlend == "off" || argBlend == "on" || argMute == "on" || argMute == "off" {
		fmt.Printf("Detected tuner: NAD %s | Power: %s\n", getDeviceID(serialPort), getState(serialPort, getPowerState))

		band := getCurrentBand(serialPort)

		if getState(serialPort, getPowerState) == "On" {
			if band == "FM" {
				fmt.Printf("FM switches (Blend: %s | Mute: %s)\n", getState(serialPort, getBlendState), getState(serialPort, getMuteState))
				showFMFrequency(serialPort)
			}
			if band == "AM" {
				showAMFrequency(serialPort)
			}
		}
	}
}

func validateFMFrequency(freq float64) error {
	if freq < minFMFrequency || freq > maxFMFrequency {
		return fmt.Errorf("frequency should be between %.2f and %.2f", minFMFrequency, maxFMFrequency)
	}
	return nil
}

func power(serialPort *serial.Port, on bool) error {
	var command []byte
	if on {
		command = turnOnPower
	} else {
		command = turnOffPower
	}
	_, err := sendCommand(serialPort, command)
	return err
}

func blend(serialPort *serial.Port, on bool) error {
	var command []byte
	if on {
		command = turnOnBlend
	} else {
		command = turnOffBlend
	}
	_, err := sendCommand(serialPort, command)
	return err
}

func mute(serialPort *serial.Port, on bool) error {
	var command []byte
	if on {
		command = turnOnMute
	} else {
		command = turnOffMute
	}
	_, err := sendCommand(serialPort, command)
	return err
}

func getState(serialPort *serial.Port, getStateFunc func(*serial.Port) (string, error)) string {
	response, err := getStateFunc(serialPort)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func openSerialPort(portName string) (*serial.Port, error) {
	config := &serial.Config{
		Name:        portName,
		Baud:        baudRate,
		ReadTimeout: readTimeout,
		Size:        dataSize,
		Parity:      parity,
		StopBits:    stopBits,
	}

	serialPort, err := serial.OpenPort(config)
	if err != nil {
		return nil, err
	}
	return serialPort, nil
}

func sendCommand(serialPort *serial.Port, command []byte) ([]byte, error) {
	serialPort.Flush()
	_, err := serialPort.Write(command)
	if err != nil {
		return nil, err
	}
	return readResponse(serialPort)
}

func readResponse(serialPort *serial.Port) ([]byte, error) {
	delimiter := byte(2)
	response := make([]byte, 0)

	for {
		buffer := make([]byte, 1)
		_, err := serialPort.Read(buffer)
		if err != nil {
			return nil, err
		}

		if buffer[0] == delimiter {
			serialPort.Flush()
			break
		}

		response = append(response, buffer[0])
	}

	_, err := serialPort.Read(response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func fmBytesToFrequency(response []byte) float64 {
	if response[3] == 94 {
		return float64(int(response[4])|int(response[5])<<8) / 100.0
	} else {
		return float64(int(response[3])|int(response[4])<<8) / 100.0
	}
}

func getDeviceID(serialPort *serial.Port) []byte {
	response, err := sendCommand(serialPort, getDevice)
	if err != nil {
		log.Fatal(err)
	}
	return response[3:7]
}

func showFMFrequency(serialPort *serial.Port) {
	response, err := sendCommand(serialPort, getFMFrequency)
	if err != nil {
		log.Fatal(err)
	}

	frequency := fmBytesToFrequency(response)
	fmt.Printf("FM Frequency: %.2f Mhz\n", frequency)
}

func showAMFrequency(serialPort *serial.Port) {
	response, err := sendCommand(serialPort, getAMFrequency)
	if err != nil {
		log.Fatal(err)
	}

	frequency := amBytesToFrequency(response)
	fmt.Printf("AM Frequency: %d Khz\n", frequency)

}

func amBytesToFrequency(bytesResponse []byte) int {
	var freqBytes []byte

	if bytesResponse[4] == 94 {
		freqBytes = []byte{bytesResponse[3], bytesResponse[5] - 64}
	} else {
		freqBytes = []byte{bytesResponse[4] - 64, bytesResponse[6] - 64}
	}

	frequency := int(freqBytes[1])<<8 | int(freqBytes[0])
	return frequency
}

func switchToFMMode(serialPort *serial.Port) error {
	_, err := sendCommand(serialPort, setFMBand)
	time.Sleep(2 * time.Second)
	return err
}

func setFMFrequency(serialPort *serial.Port, freq float64) {
	if err := validateFMFrequency(freq); err != nil {
		log.Fatal(err)
	}

	var currentBand string = getCurrentBand(serialPort)
	if currentBand != "FM" {
		switchToFMMode(serialPort)
	}

	currentFrequency, err := getCurrentFMFrequency(serialPort)
	if err != nil {
		log.Fatal(err)
	}

	if currentFrequency == freq {
	} else {
		err = setFrequency(serialPort, freq)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getPowerState(serialPort *serial.Port) (string, error) {
	response, err := sendCommand(serialPort, getPower)
	if err != nil {
		return "", err
	}

	result, err := processResponse(int(response[4]))
	if err != nil {
		return "", err
	} else {
		return result, nil
	}

}

func processResponse(response int) (string, error) {
	switch response {
	case 64:
		return "Off", nil
	case 65:
		return "On", nil
	default:
		return "Unknown", nil
	}
}

func getBlendState(serialPort *serial.Port) (string, error) {
	response, err := sendCommand(serialPort, getBlend)
	if err != nil {
		return "", err
	}

	result, err := processResponse(int(response[4]))
	if err != nil {
		return "", err
	} else {
		return result, nil
	}
}

func getMuteState(serialPort *serial.Port) (string, error) {
	response, err := sendCommand(serialPort, getFMMute)
	if err != nil {
		return "", err
	}

	result, err := processResponse(int(response[4]))
	if err != nil {
		return "", err
	} else {
		return result, nil
	}
}

func getCurrentBand(serialPort *serial.Port) string {
	response, err := sendCommand(serialPort, getBand)
	if err != nil {
		return ""
	}

	switch response[4] {
	case 64:
		return "AM"
	case 65:
		return "FM"
	default:
		return "Unknown"
	}
}

func getCurrentFMFrequency(serialPort *serial.Port) (float64, error) {
	response, err := sendCommand(serialPort, getFMFrequency)
	if err != nil {
		return 0, err
	}

	return fmBytesToFrequency(response), nil
}

func setFrequency(serialPort *serial.Port, freq float64) error {
	bytesData := fmFrequencyToBytes(freq)
	command := []byte{1, 21, 45, bytesData[0], bytesData[1]}
	crc := crcCalculate(command)

	var sendCommand []byte
	if bytesData[0] == 94 {
		command = []byte{1, 21, 45, bytesData[0], bytesData[0], bytesData[1]}
		sendCommand = append(append(command, 2), crc...)
	} else if bytesData[0] == 60 || bytesData[0] == 58 || bytesData[0] == 56 || bytesData[0] == 54 {
		command = []byte{1, 21, 45, bytesData[0], bytesData[1]}
		sendCommand = append(append(append(command, 2), crc...), 94)
	} else if bytesData[0] == 2 {
		command = []byte{1, 21, 45, 94, bytesData[0] + 64, bytesData[1]}
		sendCommand = append(append(command, 2), crc...)
	} else {
		sendCommand = append(append(command, 2), crc...)
	}

	_, err := serialPort.Write(sendCommand)
	return err
}

func fmFrequencyToBytes(frequency float64) []byte {
	freqInt := int(frequency * 100)
	freqBytes := make([]byte, 2)
	freqBytes[0] = byte(freqInt & 0xFF)
	freqBytes[1] = byte((freqInt >> 8) & 0xFF)
	return freqBytes
}

func crcCalculate(crcCommand []byte) []byte {
	var crcCalc byte
	for _, val := range crcCommand {
		crcCalc += val
	}
	crcCalc = (^crcCalc) + 1
	return []byte{crcCalc}
}
