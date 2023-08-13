package main

func crcCalculate(crcCommand []byte) []byte {
	var crcCalc byte
	for _, val := range crcCommand {
		crcCalc += val
	}
	crcCalc = (^crcCalc) + 1
	return []byte{crcCalc}
}
