package utils

import (
	"strconv"
	"strings"
)

func AbsInt(v int) int {
	if v < 0 {
		return v * -1
	}
	return v
}

func ParseIntWithFallback(value string, fallback int) int {
	value = strings.TrimSpace(value)
	intValue, err := strconv.Atoi(value)

	if err != nil {
		return fallback
	}
	return AbsInt(intValue)
}
