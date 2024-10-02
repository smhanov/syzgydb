package syzgydb

import "math"

func quantize(value float64, bits int) uint64 {
	switch bits {
	case 32:
		return uint64(math.Float32bits(float32(value)))
	case 64:
		return math.Float64bits(value)
	}
	// Ensure the value is within the expected range
	if value < -1 {
		value = -1
	} else if value > 1 {
		value = 1
	}

	// Map the float64 value from [-1, 1] to [0, maxInt]
	maxInt := (1 << bits) - 1
	quantizedValue := (value + 1) / 2 * float64(maxInt)
	return uint64(math.Round(quantizedValue))
}

func dequantize(value uint64, bits int) float64 {
	switch bits {
	case 32:
		return float64(math.Float32frombits(uint32(value)))
	case 64:
		return math.Float64frombits(value)
	}

	// Map the integer value from [0, maxInt] back to [-1, 1]
	maxInt := (1 << bits) - 1
	return (float64(value) / float64(maxInt)) * 2 - 1
}
