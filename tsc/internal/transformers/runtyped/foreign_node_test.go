package runtyped

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

// findInitializers walks the AST and returns the first numeric and string
// literal initializer expressions found.
func findInitializers(node *ast.Node, numLit, strLit **ast.Node) {
	if *numLit != nil && *strLit != nil {
		return
	}
	if node.Kind == ast.KindVariableDeclaration {
		init := node.AsVariableDeclaration().Initializer
		if init != nil {
			if init.Kind == ast.KindNumericLiteral && *numLit == nil {
				*numLit = init
			}
			if init.Kind == ast.KindStringLiteral && *strLit == nil {
				*strLit = init
			}
		}
	}
	node.ForEachChild(func(child *ast.Node) bool {
		findInitializers(child, numLit, strLit)
		return false
	})
}

// TestConvertStackNodeToExpression pins the contract that stack entry nodes
// originating in ANOTHER source file are never reused as-is in reflection data.
// Foreign pos/end values embedded into the current file's AST make the printer
// slice the current file's text with the other file's positions — a fatal
// "slice bounds out of range" panic in the native port (seen compiling loom
// against lib.dom.d.ts). Upstream guards the same hazard via
// NodeConverter.toExpression (reflection-ast.ts); convertStackNodeToExpression
// is its port.
func TestConvertStackNodeToExpression(t *testing.T) {
	t.Parallel()

	foreign := parser.ParseSourceFile(
		ast.SourceFileParseOptions{FileName: "/foreign.d.ts"},
		"const foreignNumber = 42;\nconst foreignString = \"hello\";\n",
		core.ScriptKindTS,
	)
	var numLit, strLit *ast.Node
	findInitializers(foreign.AsNode(), &numLit, &strLit)
	if numLit == nil || strLit == nil {
		t.Fatal("expected to find numeric and string literal initializers in foreign source")
	}

	tc := newTypeCompiler(printer.NewNodeFactory(printer.NewEmitContext()), nil, nil)

	// Foreign source nodes must be re-created fresh with synthetic locations.
	converted := tc.convertStackNodeToExpression(numLit)
	if converted == numLit {
		t.Fatal("foreign numeric literal node was reused as-is; expected a fresh synthesized node")
	}
	if converted.Kind != ast.KindNumericLiteral || converted.Text() != numLit.Text() {
		t.Fatalf("converted numeric literal mismatch: kind=%v text=%q", converted.Kind, converted.Text())
	}
	if !ast.PositionIsSynthesized(converted.Pos()) || !ast.PositionIsSynthesized(converted.End()) {
		t.Fatalf("converted numeric literal carries non-synthetic positions: pos=%d end=%d", converted.Pos(), converted.End())
	}

	convertedStr := tc.convertStackNodeToExpression(strLit)
	if convertedStr == strLit {
		t.Fatal("foreign string literal node was reused as-is; expected a fresh synthesized node")
	}
	if convertedStr.Kind != ast.KindStringLiteral || convertedStr.Text() != strLit.Text() {
		t.Fatalf("converted string literal mismatch: kind=%v text=%q", convertedStr.Kind, convertedStr.Text())
	}

	// An already-synthesized node (no source range, no parent) is reused as-is,
	// matching upstream's converter.
	fresh := tc.factory.NewNumericLiteral("7", ast.TokenFlagsNone)
	if got := tc.convertStackNodeToExpression(fresh); got != fresh {
		t.Fatal("synthesized node should be reused as-is")
	}
}
