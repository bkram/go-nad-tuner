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
	baudRate           = 9600
	readTimeout        = time.Second
	dataSize           = 8
	parity             = serial.ParityNone
	stopBits           = serial.Stop1
	defaultPort        = "/dev/ttyUSB0"
	minFMFrequency     = 87.50
	maxFMFrequency     = 108.00
	minAMFrequency     = 531
	maxAMFrequency     = 1602
	aMFrequencyspacing = 9
)

var (
	getBand        = []byte{1, 20, 43}
	getBlend       = []byte{1, 20, 49}
	getDevice      = []byte{1, 20, 20}
	getFMFrequency = []byte{1, 20, 45}
	getAMFrequency = []byte{1, 20, 44}
	getFMMute      = []byte{1, 20, 47}
	getPower       = []byte{1, 20, 21}
	setAMBand      = []byte{1, 21, 43, 0}
	setFMBand      = []byte{1, 21, 43, 1}
	turnOffBlend   = []byte{1, 21, 49, 0}
	turnOffMute    = []byte{1, 21, 47, 0}
	turnOffPower   = []byte{1, 21, 21, 0}
	turnOnBlend    = []byte{1, 21, 49, 1}
	turnOnMute     = []byte{1, 21, 47, 1}
	turnOnPower    = []byte{1, 21, 21, 1}
)

func main() {
	fmt.Println("nad-tuner")

	var (
		fmfrequencyPtr = flag.Float64("fm", 0, "Specify the frequency (e.g., 96.80)")
		amfrequencyPtr = flag.Int("am", 0, "Specify the frequency (e.g., 1008)")
		showPtr        = flag.Bool("show", false, "Show power state, band, and frequency")
		powerPtr       = flag.String("power", "", "Turn power on or off")
		blendPtr       = flag.String("blend", "", "Turn blend on or off")
		mutePtr        = flag.String("mute", "", "Turn mute on or off")
		portPtr        = flag.String("port", defaultPort, "Serial port name")
		bandPtr        = flag.String("band", "", "Switch band (e.g. am, fm)")
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if !(*showPtr || *powerPtr != "" || *blendPtr != "" || *mutePtr != "" ||
		*fmfrequencyPtr > 0 || *amfrequencyPtr > 0 || *bandPtr != "") {
		flag.Usage()
		return
	}

	serialPort, err := openSerialPort(*portPtr)
	if err != nil {
		log.Fatal(err)
	}
	defer serialPort.Close()

	handleCommands(serialPort, *powerPtr, *blendPtr, *mutePtr, *showPtr, *fmfrequencyPtr, *amfrequencyPtr, *bandPtr)
}

func handleCommands(serialPort *serial.Port, argPower string, argBlend string, argMute string, argShow bool,
	argFMFrequency float64, argAMFrequency int, argBand string) {
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

	if argFMFrequency > 0 {
		setFMFrequency(serialPort, argFMFrequency)
	}

	if argAMFrequency > 0 {
		setAMFrequency(serialPort, argAMFrequency)
	}

	if argBand == "fm" {
		switchToFMMode(serialPort)
	} else if argBand == "am" {
		switchToAMMode(serialPort)
	}

	if argShow || argFMFrequency > 0 || argBlend == "off" || argBlend == "on" || argMute == "on" ||
		argMute == "off" || argBand != "" || argPower == "on" {
		fmt.Printf("Detected tuner: NAD %s | Power: %s\n", getDeviceID(serialPort),
			getState(serialPort, getPowerState))

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

func validateAMFrequency(freq int) error {
	if freq < minAMFrequency || freq > maxAMFrequency {
		return fmt.Errorf("frequency should be between %d and %d kHz", minAMFrequency, maxAMFrequency)
	}
	if (freq-minAMFrequency)%aMFrequencyspacing != 0 {
		return fmt.Errorf("frequency spacing should be %d kHz", aMFrequencyspacing)
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
	crc := crcCalculate(command)
	command = append(command, 2)
	command = append(command, crc...)

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

func switchToFMMode(serialPort *serial.Port) error {
	_, err := sendCommand(serialPort, setFMBand)
	time.Sleep(2 * time.Second)
	return err
}

func switchToAMMode(serialPort *serial.Port) error {
	_, err := sendCommand(serialPort, setAMBand)
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
		err = setTunerFMFrequency(serialPort, freq)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func setAMFrequency(serialPort *serial.Port, freq int) {
	if err := validateAMFrequency(freq); err != nil {
		log.Fatal(err)
	}

	var currentBand string = getCurrentBand(serialPort)
	if currentBand != "AM" {
		switchToAMMode(serialPort)
	}

	currentFrequency, err := getCurrentAMFrequency(serialPort)
	if err != nil {
		log.Fatal(err)
	}

	if currentFrequency == freq {
	} else {
		err = setTunerAMFrequency(serialPort, freq)
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

func getCurrentAMFrequency(serialPort *serial.Port) (int, error) {
	response, err := sendCommand(serialPort, getAMFrequency)
	if err != nil {
		return 0, err
	}

	return amBytesToFrequency(response), nil
}

func setTunerFMFrequency(serialPort *serial.Port, freq float64) error {
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

func setTunerAMFrequency(serialPort *serial.Port, freq int) error {
	bytesData := amFrequencyToBytes(freq)
	command := []byte{1, 21, 44, bytesData[0], bytesData[1]}
	crc := crcCalculate(command)

	if bytesData[1] == 2 {
		command = []byte{1, 21, 44, bytesData[0], 94, bytesData[1] + 64}
	}

	command = append(command, 2)
	command = append(command, crc...)

	// var integerValues []int
	// for _, b := range command {
	// 	integerValues = append(integerValues, int(b))
	// }

	// fmt.Println("Byte values command:", integerValues)
	// fmt.Println("NAD output command:  [1 21 44 46 94 66 2 142]")

	_, err := serialPort.Write(command)
	return err
}
