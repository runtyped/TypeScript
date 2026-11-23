package runtyped

import (
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
)

// ParseJSDocAttributeFromText extracts the content of a JSDoc attribute tag
// (e.g. `@attr attr-content`) from a JSDoc comment's raw text.
//
// Port of the JS reference's parseJSDocAttributeFromText
// (packages/type-compiler/src/reflection-ast.ts). Deliberately regex-free,
// like the original.
//
// Returns (content, true) when the attribute is found; ("", false) when the
// attribute does not appear at all; ("", true) when it appears with no content.
func ParseJSDocAttributeFromText(comment string, attribute string) (string, bool) {
	tag := "@" + attribute

	index := strings.Index(comment, tag+" ")
	if index == -1 {
		start := 0
		for {
			withoutContent := strings.Index(comment[start:], tag)
			if withoutContent == -1 {
				return "", false
			}
			withoutContent += start
			// make sure next character is space, newline, tab, or end of comment
			nextIdx := withoutContent + len(attribute) + 1
			if nextIdx >= len(comment) {
				return "", true
			}
			nextCharacter := comment[nextIdx]
			if nextCharacter == ' ' || nextCharacter == '\n' || nextCharacter == '\r' || nextCharacter == '\t' {
				return "", true
			}
			start = nextIdx
		}
	}

	start := index + len(attribute) + 2
	// end is either next attribute @ or end of comment.
	nextAttribute := strings.Index(comment[start:], "@")
	endOfComment := strings.Index(comment[start:], "*/")
	var end int
	switch {
	case nextAttribute == -1 && endOfComment == -1:
		end = len(comment)
	case nextAttribute == -1:
		end = start + endOfComment
	case endOfComment == -1:
		end = start + nextAttribute
	default:
		if nextAttribute < endOfComment {
			end = start + nextAttribute
		} else {
			end = start + endOfComment
		}
	}
	content := strings.TrimSpace(comment[start:end])

	// make sure multiline comments are supported: each line is trimmed and
	// leading `\s*\*` removed.
	lines := strings.Split(content, "\n")
	for i, v := range lines {
		if _, afterStar, found := strings.Cut(v, "*"); found {
			lines[i] = strings.TrimSpace(afterStar)
		} else {
			lines[i] = strings.TrimSpace(v)
		}
	}
	return strings.Join(lines, "\n"), true
}

// commentRangeFactory is only used to build comment ranges, which needs no
// node factory state, so a shared zero value is safe across goroutines.
var commentRangeFactory ast.NodeFactory

// explicitReflectionMode is a port of compiler.ts's getExplicitReflectionMode.
// It looks for a @reflection tag in the leading comments of node and then of
// each of its ancestors, up to and including the source file (whose leading
// comments are the ones at the top of the file), and stops at the first one.
// It returns (true, true) when that tag enables reflection (no value, true,
// default, enabled, 1), (false, true) when it disables it (false, disabled,
// never, no, 0), and (false, false) when there is no tag or its value is not
// recognised, in which case the tsconfig reflection mode applies.
func explicitReflectionMode(emitContext *printer.EmitContext, node *ast.Node) (enabled bool, ok bool) {
	file := ast.GetSourceFileOfNode(node)
	if file == nil {
		return false, false
	}
	for current := node; current != nil; current = parseTreeParent(emitContext, current) {
		value, found := ParseJSDocAttributeFromText(leadingComments(file, current), "reflection")
		if !found {
			continue
		}
		switch value {
		case "", "true", "default", "enabled", "1":
			return true, true
		case "false", "disabled", "never", "no", "0":
			return false, true
		}
		return false, false
	}
	return false, false
}

// isReflectedByMode combines a file's reflection mode with the explicit
// @reflection tag of a declaration, as compiler.ts's isWithReflection does: a
// tag disabling reflection always wins, and "explicit" mode needs a tag that
// enables it.
func isReflectedByMode(emitContext *printer.EmitContext, mode string, declaration *ast.Node) bool {
	enabled, ok := explicitReflectionMode(emitContext, declaration)
	if ok && !enabled {
		return false
	}
	switch mode {
	case "never":
		return false
	case "explicit":
		return ok && enabled
	default:
		return true
	}
}

// leadingComments returns the text of the comments leading node in file,
// joined by newlines, like reflection-ast.ts's getCommentOfNode.
func leadingComments(file *ast.SourceFile, node *ast.Node) string {
	text := file.Text()
	pos := node.Pos()
	if pos < 0 || pos > len(text) {
		return ""
	}
	var comments strings.Builder
	for comment := range scanner.GetLeadingCommentRanges(&commentRangeFactory, text, pos) {
		if comments.Len() > 0 {
			comments.WriteByte('\n')
		}
		comments.WriteString(text[comment.Pos():comment.End()])
	}
	return comments.String()
}

// jsdocAttribute returns the content of the given JSDoc attribute in the
// comments leading node, like reflection-ast.ts's extractJSDocAttribute, or
// "" if there is none.
func jsdocAttribute(node *ast.Node, attribute string) string {
	if node == nil {
		return ""
	}
	file := ast.GetSourceFileOfNode(node)
	if file == nil {
		return ""
	}
	value, _ := ParseJSDocAttributeFromText(leadingComments(file, node), attribute)
	return value
}
