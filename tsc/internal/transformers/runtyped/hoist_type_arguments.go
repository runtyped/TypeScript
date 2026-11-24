package runtyped

import (
	"strconv"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/scanner"
)

// Hoisting of call-site type arguments.
//
// A call such as `is<User>(data)` passes its type argument as a packed array,
// e.g. `is(data, [() => __ΩUser, "n!"])`. Emitted inline, that array is a new
// object on every call, so the runtime can never cache what it resolved it to:
// simple references go through a few checks per call, and inline types such as
// `is<{ a: string }>(data)` run the whole type VM (and rebuild their JIT code)
// on every call.
//
// When a type argument only refers to module-level or global declarations (or
// to names it declares itself, such as mapped type keys), its packed array is
// the same on every call and is hoisted into a module-level variable:
//
//	var __ΩΩ0;
//	...
//	is(data, __ΩΩ0 || (__ΩΩ0 = [() => __ΩUser, "n!"]));
//
// The variable is initialized lazily at the call site rather than up front, so
// that calls made before the module body ran (circular imports) keep working.
// The packed array itself is unchanged, so this is invisible to runtimes that
// do not cache on it.

// hoistedTypeArgumentPrefix prefixes the module-level variables holding
// hoisted type arguments.
const hoistedTypeArgumentPrefix = "__ΩΩ"

// hoistTypeArgument returns the expression to pass for typeArg at a call site:
// a lazily initialized module-level variable holding typeExpr when typeArg can
// be hoisted, typeExpr itself otherwise.
func (tx *reflectionTransformer) hoistTypeArgument(typeArg *ast.Node, typeExpr *ast.Node) *ast.Node {
	if typeExpr == nil || typeExpr.Kind != ast.KindArrayLiteralExpression {
		return typeExpr
	}
	original := tx.EmitContext().MostOriginal(typeArg)
	if original == nil || ast.NodeIsSynthesized(original) || !tx.isHoistableTypeArgument(original) {
		return typeExpr
	}

	// Equal type arguments mean the same type anywhere in the file, since they
	// only refer to module-level names: share one variable per spelling.
	key := scanner.GetSourceTextOfNodeFromSourceFile(tx.sourceFile, original, false)
	name, ok := tx.hoistedTypeArgumentNames[key]
	if !ok {
		name = tx.nextHoistedTypeArgumentName()
		tx.hoistedTypeArgumentNames[key] = name
		tx.hoistedTypeArgumentOrder = append(tx.hoistedTypeArgumentOrder, name)
	}

	f := tx.Factory()
	// name || (name = typeExpr)
	return f.NewParenthesizedExpression(f.NewBinaryExpression(nil,
		f.NewIdentifier(name), nil,
		f.NewToken(ast.KindBarBarToken),
		f.NewParenthesizedExpression(f.NewBinaryExpression(nil,
			f.NewIdentifier(name), nil,
			f.NewToken(ast.KindEqualsToken),
			typeExpr,
		)),
	))
}

// nextHoistedTypeArgumentName returns a variable name that no identifier of
// the source file uses.
func (tx *reflectionTransformer) nextHoistedTypeArgumentName() string {
	for {
		name := hoistedTypeArgumentPrefix + strconv.Itoa(tx.hoistedTypeArgumentCounter)
		tx.hoistedTypeArgumentCounter++
		if !tx.sourceFile.HasIdentifier(name) {
			return name
		}
	}
}

// hoistedTypeArgumentDeclarations returns the `var` statement declaring every
// hoisted type argument variable, or nil if there are none.
func (tx *reflectionTransformer) hoistedTypeArgumentDeclarations() *ast.Node {
	if len(tx.hoistedTypeArgumentOrder) == 0 {
		return nil
	}
	f := tx.Factory()
	decls := make([]*ast.Node, 0, len(tx.hoistedTypeArgumentOrder))
	for _, name := range tx.hoistedTypeArgumentOrder {
		decls = append(decls, f.NewVariableDeclaration(f.NewIdentifier(name), nil, nil, nil))
	}
	return f.NewVariableStatement(nil, f.NewVariableDeclarationList(f.NewNodeList(decls), ast.NodeFlagsNone))
}

// isHoistableTypeArgument reports whether every name typeArg refers to is
// declared at module level, globally, or within typeArg itself. Names declared
// in an enclosing function, class, block or namespace (type parameters, local
// types, local classes) are only visible there, and `this` depends on the
// call's context. The check errs on the side of not hoisting: property names
// that happen to match a local declaration also prevent it.
func (tx *reflectionTransformer) isHoistableTypeArgument(typeArg *ast.Node) bool {
	hoistable := true
	var visit func(n *ast.Node) bool
	visit = func(n *ast.Node) bool {
		switch n.Kind {
		case ast.KindThisType, ast.KindThisKeyword:
			hoistable = false
			return true
		case ast.KindIdentifier:
			if !tx.isModuleLevelName(n, typeArg) {
				hoistable = false
				return true
			}
		}
		return n.ForEachChild(visit)
	}
	if visit(typeArg) {
		return false
	}
	return hoistable
}

// isModuleLevelName reports whether identifier id, found within typeArg,
// resolves to a module-level or global declaration or to one inside typeArg.
func (tx *reflectionTransformer) isModuleLevelName(id *ast.Node, typeArg *ast.Node) bool {
	name := id.AsIdentifier().Text
	for current := tx.parseTreeParent(id); current != nil; current = tx.parseTreeParent(current) {
		if current.Kind == ast.KindSourceFile {
			return true
		}
		// Class type parameters are bound into the class symbol's members
		// rather than its locals.
		if current.Kind == ast.KindClassDeclaration || current.Kind == ast.KindClassExpression {
			for _, tp := range current.TypeParameters() {
				if getNameAsString(tp.Name()) == name {
					return false
				}
			}
		}
		// Read-only accessor: ast.GetLocals lazily creates the table and races
		// with other files being emitted concurrently.
		if locals := current.Locals(); locals != nil {
			if sym, ok := locals.Lookup(name); ok && sym != nil {
				return tx.isWithin(current, typeArg)
			}
		}
	}
	return true
}

// isWithin reports whether node is ancestor or one of its descendants.
func (tx *reflectionTransformer) isWithin(node *ast.Node, ancestor *ast.Node) bool {
	for current := node; current != nil; current = tx.parseTreeParent(current) {
		if current == ancestor {
			return true
		}
		if current.Kind == ast.KindSourceFile {
			return false
		}
	}
	return false
}
