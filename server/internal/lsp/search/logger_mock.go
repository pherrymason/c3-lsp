package search

import (
	"fmt"

	log "github.com/tliron/commonlog"
)

//
// MockLogger
//

var MOCK_LOGGER MockLogger

// [Logger] that does nothing.
type MockLogger struct {
	tracker map[string][]string
}

// ([Logger] interface)
func (m MockLogger) AllowLevel(level log.Level) bool {
	return false
}

// ([Logger] interface)
func (m MockLogger) SetMaxLevel(level log.Level) {
}

// ([Logger] interface)
func (m MockLogger) GetMaxLevel() log.Level {
	return log.None
}

// ([Logger] interface)
func (m MockLogger) NewMessage(level log.Level, depth int, keysAndValues ...any) log.Message {
	return nil
}

// ([Logger] interface)
func (m MockLogger) Log(level log.Level, depth int, message string, keysAndValues ...any) {
}

// ([Logger] interface)
func (m MockLogger) Logf(level log.Level, depth int, format string, args ...any) {
}

// ([Logger] interface)
func (m MockLogger) Critical(message string, keysAndValues ...any) {
}

// ([Logger] interface)
func (m MockLogger) Criticalf(format string, args ...any) {
}

// ([Logger] interface)
func (m MockLogger) Error(message string, keysAndValues ...any) {
}

// ([Logger] interface)
func (m MockLogger) Errorf(format string, args ...any) {
}

// ([Logger] interface)
func (m MockLogger) Warning(message string, keysAndValues ...any) {
}

// ([Logger] interface)
func (m MockLogger) Warningf(format string, args ...any) {
}

// ([Logger] interface)
func (m MockLogger) Notice(message string, keysAndValues ...any) {
}

// ([Logger] interface)
func (m MockLogger) Noticef(format string, args ...any) {
}

// ([Logger] interface)
func (m MockLogger) Info(message string, keysAndValues ...any) {
}

// ([Logger] interface)
func (m MockLogger) Infof(format string, args ...any) {
}

// ([Logger] interface)
func (m MockLogger) Debug(message string, keysAndValues ...any) {
	// Extract the "message" KV value if present (structured logging format).
	tracked := message
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if keysAndValues[i] == "message" {
			tracked = fmt.Sprint(keysAndValues[i+1])
			break
		}
	}
	m.tracker["debug"] = append(m.tracker["debug"], tracked)
}

// ([Logger] interface)
func (m MockLogger) Debugf(format string, args ...any) {
}
