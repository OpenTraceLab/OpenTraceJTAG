package chain

import (
	"fmt"
	"strings"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/bsdl"
)

func parseIDCode(binary string) (uint32, uint32, error) {
	val, mask, _ := bsdl.ParseBinaryString(binary)
	if countBinaryDigits(binary) != 32 {
		return 0, 0, fmt.Errorf("chain: IDCODE must be 32 bits")
	}
	if mask == 0 {
		return 0, 0, fmt.Errorf("chain: IDCODE mask is zero")
	}
	return val, mask, nil
}

func boolsToBytes(bits []bool) []byte {
	if len(bits) == 0 {
		return nil
	}
	out := make([]byte, (len(bits)+7)/8)
	for i, bit := range bits {
		if bit {
			out[i/8] |= 1 << (uint(i) % 8)
		}
	}
	return out
}

func bytesToBools(buf []byte, bits int) []bool {
	if bits == 0 {
		return nil
	}
	out := make([]bool, bits)
	for i := 0; i < bits; i++ {
		out[i] = buf[i/8]&(1<<(uint(i)%8)) != 0
	}
	return out
}

func bitsToUint32(bits []bool) uint32 {
	var val uint32
	for i, bit := range bits {
		if bit {
			val |= 1 << uint(i)
		}
	}
	return val
}

func opcodeToBits(opcode string, width int) ([]bool, error) {
	clean := cleanBinaryString(opcode)
	if width == 0 {
		width = len(clean)
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("chain: empty opcode string")
	}
	bits := make([]bool, width)
	for i := 0; i < width; i++ {
		idx := len(clean) - 1 - i
		char := byte('0')
		if idx >= 0 {
			char = clean[idx]
		}
		switch char {
		case '0':
			bits[i] = false
		case '1':
			bits[i] = true
		default:
			return nil, fmt.Errorf("chain: invalid opcode digit %q", char)
		}
	}
	return bits, nil
}

func cleanBinaryString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '0', '1':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func countBinaryDigits(s string) int {
	count := 0
	for _, r := range s {
		switch r {
		case '0', '1', 'X', 'x':
			count++
		}
	}
	return count
}
