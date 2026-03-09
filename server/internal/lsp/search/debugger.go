package search

import (
	"time"

	"github.com/pherrymason/c3-lsp/pkg/utils"
)

type FindDebugger struct {
	enabled  bool
	depth    int
	maxDepth int
	deadline time.Time
}

func NewFindDebugger(enabled bool) FindDebugger {
	return FindDebugger{enabled: enabled, depth: 0, maxDepth: 0}
}

func (d FindDebugger) goIn() FindDebugger {
	max := utils.Max(d.depth, d.maxDepth)

	return FindDebugger{
		depth:    d.depth + 1,
		maxDepth: max,
		enabled:  d.enabled,
		deadline: d.deadline,
	}
}

func (d FindDebugger) withDeadline(deadline time.Time) FindDebugger {
	d.deadline = deadline
	return d
}

func (d FindDebugger) timedOut() bool {
	if d.deadline.IsZero() {
		return false
	}

	return time.Now().After(d.deadline)
}
