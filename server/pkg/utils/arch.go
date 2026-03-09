package utils

import (
	"runtime"
)

const (
	CPU_UNKNOWN = iota
	CPU_64      = 64
	CPU_32      = 32
)

func CpuArchitecture() uint {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return CPU_64
	case "386", "arm":
		return CPU_32
	}

	return CPU_UNKNOWN
}

func PointerSize() uint {
	arch := CpuArchitecture()

	if arch != CPU_UNKNOWN {
		return arch / 8
	}

	return 0
}
