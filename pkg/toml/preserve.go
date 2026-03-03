package toml

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	gotoml "github.com/pelletier/go-toml/v2"
)

// Marshal is a re-export of go-toml's Marshal function.
func Marshal(v interface{}) ([]byte, error) {
	return gotoml.Marshal(v)
}

// Unmarshal is a re-export of go-toml's Unmarshal function.
func Unmarshal(data []byte, v interface{}) error {
	return gotoml.Unmarshal(data, v)
}

// UpdatePreservingComments updates a TOML file while preserving comments and formatting.
// It takes the original file content, the old config (as parsed), and new config to write.
// Returns the updated content with comments preserved where possible.
func UpdatePreservingComments(originalContent []byte, oldData, newData interface{}) ([]byte, error) {
	oldBytes, newBytes, err := marshalBoth(oldData, newData)
	if err != nil {
		return nil, err
	}

	if isContentIdentical(oldBytes, newBytes) {
		return originalContent, nil
	}

	oldMap, newMap, err := parseBothToMaps(oldBytes, newBytes)
	if err != nil {
		return nil, err
	}

	return applySmartUpdate(originalContent, oldMap, newMap, newBytes)
}

func marshalBoth(oldData, newData interface{}) ([]byte, []byte, error) {
	oldBytes, err := Marshal(oldData)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling old data: %w", err)
	}

	newBytes, err := Marshal(newData)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling new data: %w", err)
	}

	return oldBytes, newBytes, nil
}

func isContentIdentical(oldBytes, newBytes []byte) bool {
	return bytes.Equal(normalizeToml(oldBytes), normalizeToml(newBytes))
}

func parseBothToMaps(oldBytes, newBytes []byte) (map[string]interface{}, map[string]interface{}, error) {
	oldMap := make(map[string]interface{})
	newMap := make(map[string]interface{})

	if err := Unmarshal(oldBytes, &oldMap); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling old: %w", err)
	}

	if err := Unmarshal(newBytes, &newMap); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling new: %w", err)
	}

	return oldMap, newMap, nil
}

func applySmartUpdate(originalContent []byte, oldMap, newMap map[string]interface{}, newBytes []byte) ([]byte, error) {
	result, err := smartUpdate(originalContent, oldMap, newMap, newBytes)
	if err != nil {
		return newBytes, nil
	}
	return result, nil
}

// normalizeToml removes blank lines and normalizes whitespace for comparison
func normalizeToml(data []byte) []byte {
	lines := bytes.Split(data, []byte("\n"))
	var normalized [][]byte

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			normalized = append(normalized, trimmed)
		}
	}

	return bytes.Join(normalized, []byte("\n"))
}

// smartUpdate attempts to surgically update specific sections
func smartUpdate(original []byte, oldMap, newMap map[string]interface{}, fullNew []byte) ([]byte, error) {
	changes := detectChanges(oldMap, newMap)

	if changes.tooComplex {
		return nil, fmt.Errorf("changes too complex for smart update")
	}

	result := original
	var err error

	for i := len(changes.sections) - 1; i >= 0; i-- {
		section := changes.sections[i]
		result, err = updateSection(result, section, newMap, fullNew)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

type sectionChanges struct {
	sections   []string
	tooComplex bool
}

func detectChanges(oldMap, newMap map[string]interface{}) sectionChanges {
	changes := sectionChanges{}

	for key := range newMap {
		if !mapsEqual(oldMap[key], newMap[key]) {
			changes.sections = append(changes.sections, key)
		}
	}

	for key := range oldMap {
		if _, exists := newMap[key]; !exists {
			changes.sections = append(changes.sections, key)
		}
	}

	return changes
}

func mapsEqual(a, b interface{}) bool {
	aBytes, err1 := Marshal(a)
	bBytes, err2 := Marshal(b)

	if err1 != nil || err2 != nil {
		return false
	}

	return bytes.Equal(normalizeToml(aBytes), normalizeToml(bBytes))
}

func updateSection(content []byte, sectionName string, newMap map[string]interface{}, fullNew []byte) ([]byte, error) {
	start, end, err := findSectionBounds(content, sectionName)
	if err != nil {
		return appendSection(content, sectionName, newMap, fullNew), nil
	}

	newSection, err := extractSectionContent(fullNew, sectionName, newMap)
	if err != nil {
		return nil, err
	}

	if _, exists := newMap[sectionName]; !exists {
		return removeSection(content, start, end), nil
	}

	// Preserve within-section comments by anchoring them to structural lines
	anchors := extractCommentAnchors(content[start:end])
	if len(anchors) > 0 {
		newSection = injectSectionComments(newSection, anchors)
	}

	// Preserve trailing comments/whitespace before next section
	trailingComments := extractTrailingComments(content[start:end])
	if len(trailingComments) > 0 {
		newSection = append(newSection, trailingComments...)
	}

	return replaceSection(content, start, end, newSection), nil
}

func findSectionBounds(content []byte, section string) (start, end int, err error) {
	headerPattern := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(section) + `\]\s*$`)

	matches := headerPattern.FindIndex(content)
	if matches == nil {
		// No main section header found, but check if there are subsections
		// e.g., no [groups] but we have [groups.web]
		subsectionPattern := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(section) + `\.[^]]+\]\s*$`)
		subsectionMatches := subsectionPattern.FindIndex(content)
		if subsectionMatches == nil {
			return 0, 0, fmt.Errorf("section not found")
		}

		// Found a subsection, so use it as the starting point
		matches = subsectionMatches
	}

	start = matches[0]

	// Find next section that is NOT a subsection of the current section
	// We need to skip [section.xxx] but stop at [other] or [section-other]
	end = findNextNonSubsection(content[matches[1]:], section)
	if end == -1 {
		end = len(content)
	} else {
		end = matches[1] + end
	}

	return start, end, nil
}

func findNextNonSubsection(content []byte, section string) int {
	allSectionPattern := regexp.MustCompile(`(?m)^\[([^]]+)\]\s*$`)
	subsectionPrefix := section + "."

	matches := allSectionPattern.FindAllStringSubmatchIndex(string(content), -1)
	for _, match := range matches {
		// match[2] and match[3] are start and end of capture group 1 (the section name)
		sectionName := string(content[match[2]:match[3]])

		// If this section doesn't start with our prefix, it's not a subsection
		if !strings.HasPrefix(sectionName, subsectionPrefix) {
			return match[0] // Return start of the full match
		}
	}

	return -1
}

func extractSectionContent(content []byte, section string, dataMap map[string]interface{}) ([]byte, error) {
	sectionData := map[string]interface{}{
		section: dataMap[section],
	}

	marshaled, err := Marshal(sectionData)
	if err != nil {
		return nil, err
	}

	return marshaled, nil
}

func appendSection(content []byte, section string, newMap map[string]interface{}, fullNew []byte) []byte {
	newSection, err := extractSectionContent(fullNew, section, newMap)
	if err != nil {
		return append(content, fullNew...)
	}

	if len(content) > 0 && content[len(content)-1] != '\n' {
		content = append(content, '\n')
	}

	if len(content) > 0 {
		content = append(content, '\n')
	}

	return append(content, newSection...)
}

func removeSection(content []byte, start, end int) []byte {
	for end < len(content) && content[end] == '\n' {
		end++
		if end < len(content) && content[end] == '\n' {
			break
		}
	}

	return append(content[:start], content[end:]...)
}

func replaceSection(content []byte, start, end int, newSection []byte) []byte {
	result := make([]byte, 0, len(content)-end+start+len(newSection))
	result = append(result, content[:start]...)
	result = append(result, newSection...)

	if end < len(content) && content[end-1] != '\n' && len(newSection) > 0 && newSection[len(newSection)-1] != '\n' {
		result = append(result, '\n')
	}

	result = append(result, content[end:]...)
	return result
}

// commentAnchor associates a block of comment lines with the structural line
// (subsection header or key=value) that immediately follows them.
type commentAnchor struct {
	comments []string
	identity string
}

// extractCommentAnchors parses a TOML section and returns comments paired with
// the identity of the structural line they precede. This allows comments to be
// re-inserted at the correct position when the section is regenerated.
func extractCommentAnchors(sectionContent []byte) []commentAnchor {
	var anchors []commentAnchor
	var pending []string
	currentSubsection := ""

	lines := strings.Split(string(sectionContent), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if len(pending) > 0 {
				pending = append(pending, line)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			pending = append(pending, line)
			continue
		}

		// Track current subsection for scoped key identity.
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSubsection = strings.Trim(trimmed, "[] \t")
		}

		if len(pending) > 0 {
			id := anchorIdentity(trimmed, currentSubsection)
			if id != "" {
				cleaned := trimTrailingBlankStrings(pending)
				if len(cleaned) > 0 {
					anchors = append(anchors, commentAnchor{
						comments: cleaned,
						identity: id,
					})
				}
			}
			pending = nil
		}
	}

	return anchors
}

// anchorIdentity returns a stable identity for a structural TOML line.
// Subsection headers and key-value pairs under the same parent use a dot-separated
// form so that inline tables (key = { ... }) and subsection notation ([parent.key])
// produce the same identity.
func anchorIdentity(line, currentSubsection string) string {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return strings.Trim(trimmed, "[] \t")
	}

	if idx := strings.Index(trimmed, "="); idx > 0 {
		key := strings.TrimSpace(trimmed[:idx])
		if currentSubsection != "" {
			return currentSubsection + "." + key
		}
		return key
	}

	return ""
}

func trimTrailingBlankStrings(lines []string) []string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[:end]
}

// injectSectionComments re-inserts comment anchors into regenerated section content.
func injectSectionComments(newSection []byte, anchors []commentAnchor) []byte {
	if len(anchors) == 0 {
		return newSection
	}

	lookup := make(map[string][]string, len(anchors))
	for _, a := range anchors {
		lookup[a.identity] = a.comments
	}

	var result bytes.Buffer
	currentSubsection := ""

	scanner := bufio.NewScanner(bytes.NewReader(newSection))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSubsection = strings.Trim(trimmed, "[] \t")
		}

		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			id := anchorIdentity(trimmed, currentSubsection)
			if comments, ok := lookup[id]; ok {
				for _, c := range comments {
					result.WriteString(c)
					result.WriteString("\n")
				}
				delete(lookup, id)
			}
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.Bytes()
}

// extractTrailingComments extracts comments and whitespace from the end of a section.
// This preserves formatting between sections.
func extractTrailingComments(sectionContent []byte) []byte {
	lines := bytes.Split(sectionContent, []byte("\n"))

	// Find where actual content ends (working backwards)
	lastContentLine := -1
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := bytes.TrimSpace(lines[i])
		// Skip empty lines and header lines
		if len(trimmed) > 0 && !bytes.HasPrefix(trimmed, []byte("[")) {
			// This is actual TOML content (not comment, not empty, not header)
			if !bytes.HasPrefix(trimmed, []byte("#")) {
				lastContentLine = i
				break
			}
		}
	}

	// If no content found, or everything after is empty, return nothing
	if lastContentLine == -1 || lastContentLine >= len(lines)-1 {
		return nil
	}

	// Extract everything after last content line
	trailing := bytes.Join(lines[lastContentLine+1:], []byte("\n"))
	if len(trailing) > 0 && !bytes.HasSuffix(trailing, []byte("\n")) {
		trailing = append(trailing, '\n')
	}

	return trailing
}

// PreserveUserEdits merges user-added comments and formatting from original into new content.
// This is a simpler fallback that preserves comments from workspace section only.
func PreserveUserEdits(original, generated []byte) []byte {
	comments := extractComments(original)
	if len(comments) == 0 {
		return generated
	}

	return reinsertComments(generated, comments)
}

func extractComments(content []byte) []string {
	var comments []string
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			comments = append(comments, line)
		}
	}

	return comments
}

func reinsertComments(content []byte, comments []string) []byte {
	if len(comments) == 0 {
		return content
	}

	var result bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		if shouldInsertComments(lineNum, line, comments) {
			writeComments(&result, comments)
			comments = nil
		}
		result.WriteString(line)
		result.WriteString("\n")
		lineNum++
	}

	return result.Bytes()
}

func shouldInsertComments(lineNum int, line string, comments []string) bool {
	return lineNum == 0 && comments != nil && strings.HasPrefix(strings.TrimSpace(line), "[")
}

func writeComments(buf *bytes.Buffer, comments []string) {
	for _, comment := range comments {
		buf.WriteString(comment)
		buf.WriteString("\n")
	}
}
