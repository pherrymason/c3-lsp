package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiagnosticsDelayFromMs(t *testing.T) {
	assert.Equal(t, 2*time.Second, diagnosticsDelayFromMs(2000))
	assert.Equal(t, 150*time.Millisecond, diagnosticsDelayFromMs(150))
}
