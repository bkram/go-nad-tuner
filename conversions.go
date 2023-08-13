package main

func fmFrequencyToBytes(frequency float64) []byte {
	freqInt := int(frequency * 100)
	freqBytes := make([]byte, 2)
	freqBytes[0] = byte(freqInt & 0xFF)
	freqBytes[1] = byte((freqInt >> 8) & 0xFF)
	return freqBytes
}

func fmBytesToFrequency(response []byte) float64 {
	if response[3] == 94 {
		return float64(int(response[4])|int(response[5])<<8) / 100.0
	} else {
		return float64(int(response[3])|int(response[4])<<8) / 100.0
	}
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
