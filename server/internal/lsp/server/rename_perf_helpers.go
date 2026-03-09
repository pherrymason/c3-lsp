package server

import (
	"sort"
	"unicode/utf16"
	"unicode/utf8"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type byteSpan struct {
	start int
	end   int
}

func buildCommentStringSpans(source string) []byteSpan {
	if source == "" {
		return nil
	}

	const (
		stateCode = iota
		stateLineComment
		stateBlockComment
		stateDoubleQuote
		stateSingleQuote
	)

	spans := make([]byteSpan, 0, 64)
	state := stateCode
	stateStart := -1
	escaped := false

	for i := 0; i < len(source); i++ {
		ch := source[i]
		next := byte(0)
		hasNext := i+1 < len(source)
		if hasNext {
			next = source[i+1]
		}

		switch state {
		case stateLineComment:
			if ch == '\n' {
				spans = append(spans, byteSpan{start: stateStart, end: i})
				state = stateCode
				stateStart = -1
			}
			continue
		case stateBlockComment:
			if ch == '*' && hasNext && next == '/' {
				spans = append(spans, byteSpan{start: stateStart, end: i + 2})
				state = stateCode
				stateStart = -1
				i++
			}
			continue
		case stateDoubleQuote:
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				spans = append(spans, byteSpan{start: stateStart, end: i + 1})
				state = stateCode
				stateStart = -1
			}
			continue
		case stateSingleQuote:
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				spans = append(spans, byteSpan{start: stateStart, end: i + 1})
				state = stateCode
				stateStart = -1
			}
			continue
		}

		if ch == '/' && hasNext && next == '/' {
			state = stateLineComment
			stateStart = i
			i++
			continue
		}
		if ch == '/' && hasNext && next == '*' {
			state = stateBlockComment
			stateStart = i
			i++
			continue
		}
		if ch == '"' {
			state = stateDoubleQuote
			stateStart = i
			continue
		}
		if ch == '\'' {
			state = stateSingleQuote
			stateStart = i
		}
	}

	if state != stateCode && stateStart >= 0 {
		spans = append(spans, byteSpan{start: stateStart, end: len(source)})
	}

	return spans
}

func byteIndexInSpans(spans []byteSpan, index int) bool {
	if len(spans) == 0 || index < 0 {
		return false
	}

	i := sort.Search(len(spans), func(i int) bool {
		return spans[i].end > index
	})
	if i >= len(spans) {
		return false
	}

	span := spans[i]
	return index >= span.start && index < span.end
}

type lspPositionCursor struct {
	content string
	index   int
	line    uint32
	char    uint32
}

func newLSPPositionCursor(content string) *lspPositionCursor {
	return &lspPositionCursor{content: content}
}

func (c *lspPositionCursor) PositionAt(index int) protocol.Position {
	if c == nil {
		return protocol.Position{}
	}

	if index < 0 {
		index = 0
	}
	if index > len(c.content) {
		index = len(c.content)
	}

	if index < c.index {
		c.index = 0
		c.line = 0
		c.char = 0
	}

	for c.index < index {
		r, w := utf8.DecodeRuneInString(c.content[c.index:])
		if r == '\n' {
			c.line++
			c.char = 0
			c.index += w
			continue
		}

		if r == utf8.RuneError && w == 1 {
			c.char++
			c.index += w
			continue
		}

		c.char += uint32(len(utf16.Encode([]rune{r})))
		c.index += w
	}

	return protocol.Position{Line: c.line, Character: c.char}
}
