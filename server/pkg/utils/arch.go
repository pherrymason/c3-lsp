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
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		return CPU_64
	} else if runtime.GOARCH == "386" || runtime.GOARCH == "arm" {
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
