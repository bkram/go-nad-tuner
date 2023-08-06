package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tarm/serial"
)

const (
	baudRate     = 9600
	readTimeout  = time.Second
	dataSize     = 8
	parity       = serial.ParityNone
	stopBits     = serial.Stop1
	defaultPort  = "/dev/ttyUSB0"
	minFrequency = 87.50
	maxFrequency = 108.00
)

var (
	getBand        = []byte{1, 20, 43, 2, 192}
	getBlend       = []byte{1, 20, 49, 2, 186}
	getFMFrequency = []byte{1, 20, 45, 2, 190}
	getFMMute      = []byte{1, 20, 47, 2, 188}
	getPower       = []byte{1, 20, 21, 2, 214}
	setFMBand      = []byte{1, 22, 129, 2, 104}
	turnOffBlend   = []byte{1, 21, 49, 94, 64, 2, 185}
	turnOffMute    = []byte{1, 21, 47, 94, 64, 2, 187}
	turnOffPower   = []byte{1, 21, 21, 0, 2, 213}
	turnOnBlend    = []byte{1, 21, 49, 94, 65, 2, 184}
	turnOnMute     = []byte{1, 21, 47, 94, 65, 2, 186}
	turnOnPower    = []byte{1, 21, 21, 1, 2, 212}
)

func main() {
	fmt.Println("nad-tuner")

	var (
		frequencyPtr = flag.Float64("fm", 0, "Specify the frequency (e.g., 96.80)")
		showPtr      = flag.Bool("show", false, "Show power state, band, and frequency")
		powerPtr     = flag.String("power", "", "Turn power on or off")
		blendPtr     = flag.String("blend", "", "Turn blend on or off")
		mutePtr      = flag.String("mute", "", "Turn mute on or off")
		portPtr      = flag.String("port", defaultPort, "Serial port name")
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if !(*showPtr || *powerPtr != "" || *blendPtr != "" || *mutePtr != "" || *frequencyPtr > 0) {
		flag.Usage()
		return
	}

	serialPort, err := openSerialPort(*portPtr)
	if err != nil {
		log.Fatal(err)
	}
	defer serialPort.Close()

	handleCommands(serialPort, *powerPtr, *blendPtr, *mutePtr, *showPtr, *frequencyPtr)
}

func handleCommands(serialPort *serial.Port, power string, blend string, mute string, show bool, frequency float64) {
	if power == "on" {
		turnOnOff(serialPort, true)
	}

	if power == "off" {
		turnOnOff(serialPort, false)
	}

	if blend == "on" {
		blendOnOff(serialPort, true)
	}

	if blend == "off" {
		blendOnOff(serialPort, false)
	}
	if mute == "on" {
		muteOnOff(serialPort, true)
	}

	if mute == "off" {
		muteOnOff(serialPort, false)
	}

	if frequency > 0 {
		setFMFrequency(serialPort, frequency)
	}

	if show || frequency > 0 || blend == "off" || blend == "on" || mute == "on" || mute == "off" {
		showFMFrequency(serialPort)
		showState(serialPort, "Power", getPowerState)
		showState(serialPort, "Blend", getBlendState)
		showState(serialPort, "Mute", getMuteState)
	}

}

func validateFrequencyFM(freq float64) error {
	if freq < minFrequency || freq > maxFrequency {
		return fmt.Errorf("frequency should be between %.2f and %.2f", minFrequency, maxFrequency)
	}
	return nil
}

func turnOnOff(serialPort *serial.Port, on bool) error {
	var command []byte
	if on {
		command = turnOnPower
	} else {
		command = turnOffPower
	}
	_, err := sendCommand(serialPort, command)
	return err
}

func blendOnOff(serialPort *serial.Port, on bool) error {
	var command []byte
	if on {
		command = turnOnBlend
	} else {
		command = turnOffBlend
	}
	_, err := sendCommand(serialPort, command)
	return err
}

func muteOnOff(serialPort *serial.Port, on bool) error {
	var command []byte
	if on {
		command = turnOnMute
	} else {
		command = turnOffMute
	}
	_, err := sendCommand(serialPort, command)
	return err
}

func showState(serialPort *serial.Port, stateName string, getStateFunc func(*serial.Port) (string, error)) {
	response, err := getStateFunc(serialPort)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %s\n", stateName, response)
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

func getFMFrequencyValue(response []byte) float64 {
	if response[3] == 94 {
		return float64(int(response[4])|int(response[5])<<8) / 100.0
	} else {
		return float64(int(response[3])|int(response[4])<<8) / 100.0
	}
}

func showFMFrequency(serialPort *serial.Port) {
	response, err := sendCommand(serialPort, getFMFrequency)
	if err != nil {
		log.Fatal(err)
	}

	currentFrequency := getFMFrequencyValue(response)
	fmt.Printf("Frequency: %.2f Mhz\n", currentFrequency)
}

func switchToFMMode(serialPort *serial.Port) error {
	_, err := sendCommand(serialPort, setFMBand)
	return err
}

func setFMFrequency(serialPort *serial.Port, freq float64) {
	if err := validateFrequencyFM(freq); err != nil {
		log.Fatal(err)
	}

	currentBand, err := getCurrentBand(serialPort)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("Band: %s\n", currentBand)

	if currentBand != "FM" {
		// fmt.Println("Switching to FM mode...")
		err = switchToFMMode(serialPort)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Println("Switched to FM mode.")
	}

	currentFrequency, err := getCurrentFrequency(serialPort)
	if err != nil {
		log.Fatal(err)
	}

	if currentFrequency == freq {
		// fmt.Printf("Frequency already set to: %.2f Mhz FM\n", freq)
	} else {
		err = setFrequency(serialPort, freq)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Printf("Frequency set to: %.2f Mhz FM\n", freq)
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

func getCurrentBand(serialPort *serial.Port) (string, error) {
	response, err := sendCommand(serialPort, getBand)
	if err != nil {
		return "", err
	}

	switch response[4] {
	case 64:
		return "AM", nil
	case 65:
		return "FM", nil
	default:
		return "Unknown", nil
	}
}

func getCurrentFrequency(serialPort *serial.Port) (float64, error) {
	response, err := sendCommand(serialPort, getFMFrequency)
	if err != nil {
		return 0, err
	}

	return getFMFrequencyValue(response), nil
}

func setFrequency(serialPort *serial.Port, freq float64) error {
	bytesData := frequencyToBytes(freq)
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

func frequencyToBytes(frequency float64) []byte {
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
