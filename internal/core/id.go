package core

import "strings"

func MakeSessionID(source Source, nativeID string) string {
	return string(source) + ":" + nativeID
}

func ParseSessionID(sessionID string) (Source, string, error) {
	sourceText, nativeID, ok := strings.Cut(sessionID, ":")
	source := Source(sourceText)
	if !ok || nativeID == "" || !IsSource(source) {
		return "", "", NewError(ErrInvalidSessionID, "invalid session id: "+sessionID)
	}
	return source, nativeID, nil
}

func IsSource(source Source) bool {
	return source == SourceCodex || source == SourceClaude || source == SourceCursor
}
