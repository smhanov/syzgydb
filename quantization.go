package syzygy

import "math"

func quantize(value float64, bits int) uint64 {
	switch bits {
	case 32:
		return uint64(math.Float32bits(float32(value)))
	case 64:
		return math.Float64bits(value)
	}
	// Implement quantization logic based on the number of bits
	// Example: map the float64 value to an integer range based on bits
	maxInt := (1 << bits) - 1
	return uint64(math.Round(value * float64(maxInt)))
}

func dequantize(value uint64, bits int) float64 {
	switch bits {
	case 32:
		return float64(math.Float32frombits(uint32(value)))
	case 64:
		return math.Float64frombits(value)
	}

	// Implement dequantization logic based on the number of bits
	maxInt := (1 << bits) - 1
	return float64(value) / float64(maxInt)
}
