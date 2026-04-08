package dockerfile

import (
	"errors"
	"strings"
)

// ErrNoCustomStage is returned when the Dockerfile does not contain
// the expected custom stage boundary line.
var ErrNoCustomStage = errors.New("dockerfile: custom stage not found (expected \"FROM agentbox AS custom\")")

// SplitAtCustomStage splits a Dockerfile into two parts:
// the agentbox-managed content (everything before the custom stage line)
// and the user-managed content (everything from the custom stage line onward, inclusive).
//
// The function uses index-based slicing on the original string to find the
// boundary, preserving all original whitespace and newlines exactly as-is.
// It scans for a line matching "FROM agentbox AS custom". Matching is
// case-insensitive on the FROM and AS keywords, whitespace-tolerant
// (leading/trailing whitespace trimmed, fields split on any whitespace),
// and exact on the stage names "agentbox" and "custom".
//
// If no match is found, it returns ErrNoCustomStage.
func SplitAtCustomStage(content string) (agentboxPart, userPart string, err error) {
	offset := 0
	for offset < len(content) {
		// Find end of current line.
		nl := strings.IndexByte(content[offset:], '\n')
		var line string
		if nl == -1 {
			line = content[offset:]
		} else {
			line = content[offset : offset+nl]
		}
		if matchesCustomStage(strings.TrimSpace(line)) {
			return content[:offset], content[offset:], nil
		}
		if nl == -1 {
			break
		}
		offset += nl + 1
	}
	return "", "", ErrNoCustomStage
}

// matchesCustomStage checks if a trimmed line matches the custom stage pattern:
// FROM agentbox AS custom (case-insensitive on FROM/AS, exact on agentbox/custom).
func matchesCustomStage(trimmed string) bool {
	fields := strings.Fields(trimmed)
	if len(fields) != 4 {
		return false
	}
	return strings.EqualFold(fields[0], "FROM") &&
		fields[1] == "agentbox" &&
		strings.EqualFold(fields[2], "AS") &&
		fields[3] == "custom"
}
