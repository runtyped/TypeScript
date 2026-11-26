package runtyped

import (
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/transformers/runtyped/config"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
)

// ReflectionConfig decides, per source file, whether and how that file is
// reflected, based on the `reflection` (and `deepkitCompilerOptions`) fields of
// tsconfig.json.
//
// When the program was created from a tsconfig.json (`tsc -p`), that file and
// its `extends` chain apply to every file in the program. Otherwise each file
// uses the nearest tsconfig.json above it. This matches
// @runtyped/type-compiler's getConfigResolver.
//
// One deliberate difference from @runtyped/type-compiler: a missing
// `reflection` field means "default" rather than "never". Compiling with this
// tsc is the opt-in, so a project without the field is fully reflected, as it
// was before tsconfig.json was consulted at all.
//
// A nil *ReflectionConfig reflects every file in "default" mode.
type ReflectionConfig struct {
	host           tsoptions.ParseConfigHost
	configFilePath string

	mu    sync.Mutex
	cache map[string]*config.ConfigResolver
}

// NewReflectionConfig returns a ReflectionConfig that reads tsconfig.json files
// from fs. It returns nil when fs is nil.
func NewReflectionConfig(fs vfs.FS, options *core.CompilerOptions) *ReflectionConfig {
	if fs == nil {
		return nil
	}
	rc := &ReflectionConfig{
		host:  tsoptions.NewDefaultParseConfigHost(fs),
		cache: make(map[string]*config.ConfigResolver),
	}
	if options != nil && options.ConfigFilePath != "" {
		rc.configFilePath = options.ConfigFilePath.AsString()
	}
	return rc
}

// ModeFor returns the reflection mode of the file at fileName.
func (rc *ReflectionConfig) ModeFor(fileName string) config.Mode {
	if rc == nil {
		return config.ModeDefault
	}
	rc.mu.Lock()
	resolver := config.GetConfigResolver(rc.cache, rc.host, nil, struct{ FileName string }{fileName}, rc.configFilePath)
	rc.mu.Unlock()

	reflection := resolver.Config.Reflection
	if reflection == nil {
		reflection = config.ModeDefault
	}
	return config.ReflectionModeMatcher(config.ReflectionConfig{
		Exclude:    resolver.Config.Exclude,
		Reflection: reflection,
	}, fileName)
}

// ReflectTypeOnlyImportsFor reports whether type-only imports in the file at
// fileName are reflected (the `reflectTypeOnlyImports` field of tsconfig.json).
// It is false without a tsconfig.json, or with a nil *ReflectionConfig.
func (rc *ReflectionConfig) ReflectTypeOnlyImportsFor(fileName string) bool {
	if rc == nil {
		return false
	}
	rc.mu.Lock()
	resolver := config.GetConfigResolver(rc.cache, rc.host, nil, struct{ FileName string }{fileName}, rc.configFilePath)
	rc.mu.Unlock()
	return resolver.Config.ReflectTypeOnlyImports
}

// fileModes resolves reflection modes for the files a transformer looks at.
// A non-empty override (an explicit TransformOptions.ReflectionMode or
// SetReflectionModeOverride) wins over tsconfig.json for every file.
type fileModes struct {
	override string
	config   *ReflectionConfig
}

func (m fileModes) modeFor(file *ast.SourceFile) string {
	if m.override != "" {
		return m.override
	}
	if file == nil {
		return string(config.ModeDefault)
	}
	return string(m.config.ModeFor(file.FileName().AsString()))
}

func (m fileModes) reflectTypeOnlyImportsFor(file *ast.SourceFile) bool {
	if file == nil {
		return false
	}
	return m.config.ReflectTypeOnlyImportsFor(file.FileName().AsString())
}
