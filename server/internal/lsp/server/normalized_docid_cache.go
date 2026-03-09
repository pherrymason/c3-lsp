package server

import (
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const normalizedDocIDCacheMaxEntries = 2048

func (s *Server) normalizedDocIDFromURI(uri protocol.DocumentUri) string {
	if s == nil {
		return utils.NormalizePath(uri)
	}

	key := string(uri)
	if key == "" {
		return ""
	}

	s.normalizedDocIDCacheMu.Lock()
	if s.normalizedDocIDCache != nil {
		if cached, ok := s.normalizedDocIDCache[key]; ok {
			s.normalizedDocIDCacheMu.Unlock()
			return cached
		}
	}
	s.normalizedDocIDCacheMu.Unlock()

	normalized := utils.NormalizePath(uri)

	s.normalizedDocIDCacheMu.Lock()
	if s.normalizedDocIDCache == nil {
		s.normalizedDocIDCache = make(map[string]string)
	}
	if len(s.normalizedDocIDCache) >= normalizedDocIDCacheMaxEntries {
		s.normalizedDocIDCache = make(map[string]string, normalizedDocIDCacheMaxEntries)
	}
	s.normalizedDocIDCache[key] = normalized
	s.normalizedDocIDCacheMu.Unlock()

	return normalized
}
