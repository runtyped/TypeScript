package runtyped

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
)

// DeclarationTransformer is a post-processor for .d.ts emit.
// It finds exported type/interface/enum declarations and appends
// `export declare type __ΩX = any[]` for each, so that consumers
// can detect the __Ω symbol and import it for cross-file type resolution.
type DeclarationTransformer struct {
	factory *printer.NodeFactory

	// Reflection mode of the source file the declarations were emitted from;
	// empty means "default".
	mode string
}

// NewDeclarationTransformer creates a new declaration transformer.
func NewDeclarationTransformer(emitContext *printer.EmitContext) *DeclarationTransformer {
	return &DeclarationTransformer{
		factory: emitContext.Factory,
	}
}

// ForSourceFile sets the reflection mode from the original (.ts) source file
// the declarations are emitted for, so that the .d.ts only advertises __Ω
// symbols the emitted JavaScript actually defines.
func (dt *DeclarationTransformer) ForSourceFile(rc *ReflectionConfig, original *ast.SourceFile) *DeclarationTransformer {
	dt.mode = fileModes{override: GetReflectionModeOverride(), config: rc}.modeFor(original)
	return dt
}

// TransformSourceFile appends __Ω type alias declarations to a .d.ts source file.
func (dt *DeclarationTransformer) TransformSourceFile(sourceFile *ast.SourceFile) *ast.SourceFile {
	// Only process declaration files
	if !sourceFile.IsDeclarationFile || dt.mode == "never" {
		return sourceFile
	}

	var addExports []string
	handled := make(map[string]bool)

	for _, stmt := range sourceFile.Statements.Nodes {
		switch stmt.Kind {
		case ast.KindTypeAliasDeclaration:
			if hasModifier(stmt, ast.ModifierFlagsExport) && stmt.Name() != nil && dt.reflects(sourceFile, stmt) {
				name := getIdentifierName(stmt.Name())
				if name != "" && !handled[name] {
					handled[name] = true
					addExports = append(addExports, name)
				}
			}
		case ast.KindInterfaceDeclaration:
			if hasModifier(stmt, ast.ModifierFlagsExport) && stmt.Name() != nil && dt.reflects(sourceFile, stmt) {
				name := getIdentifierName(stmt.Name())
				if name != "" && !handled[name] {
					handled[name] = true
					addExports = append(addExports, name)
				}
			}
		case ast.KindEnumDeclaration:
			if hasModifier(stmt, ast.ModifierFlagsExport) && stmt.Name() != nil && dt.reflects(sourceFile, stmt) {
				name := getIdentifierName(stmt.Name())
				if name != "" && !handled[name] {
					handled[name] = true
					addExports = append(addExports, name)
				}
			}
		}
	}

	if len(addExports) == 0 {
		return sourceFile
	}

	// Build: export declare type __ΩX = any[]
	var newStatements []*ast.Node
	newStatements = append(newStatements, sourceFile.Statements.Nodes...)

	for _, name := range addExports {
		omegaName := dt.factory.NewIdentifier("__Ω" + name)
		anyType := dt.factory.NewArrayTypeNode(dt.factory.NewKeywordTypeNode(ast.KindAnyKeyword))
		modifiers := dt.factory.NewModifierList([]*ast.Node{
			dt.factory.NewModifier(ast.KindExportKeyword),
			dt.factory.NewModifier(ast.KindDeclareKeyword),
		})
		typeAlias := dt.factory.NewTypeAliasDeclaration(
			modifiers,
			omegaName,
			nil, // no type parameters
			anyType,
		)
		newStatements = append(newStatements, typeAlias)
	}

	updatedStatements := dt.factory.NewNodeList(newStatements)
	updatedStatements.Loc = sourceFile.Statements.Loc
	return dt.factory.UpdateSourceFile(sourceFile, updatedStatements, sourceFile.EndOfFileToken).AsSourceFile()
}

// reflects mirrors shouldReflect: in "explicit" mode only declarations tagged
// with @reflection get a __Ω symbol.
func (dt *DeclarationTransformer) reflects(file *ast.SourceFile, stmt *ast.Node) bool {
	return dt.mode != "explicit" || hasReflectionJSDoc(file, stmt)
}
