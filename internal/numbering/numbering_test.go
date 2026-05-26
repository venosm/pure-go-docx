package numbering

import (
	"strings"
	"testing"
)

func TestResolve_Decimal_Sequential(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`)

	for _, tc := range []struct {
		name string
		want int
	}{
		{name: "first", want: 1},
		{name: "second", want: 2},
		{name: "third", want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			def, ordinal, ok := resolver.Resolve(1, 0)
			if !ok {
				t.Fatal("Resolve() ok = false, want true")
			}
			if def.Format != "decimal" {
				t.Fatalf("Format = %q, want decimal", def.Format)
			}
			if ordinal != tc.want {
				t.Fatalf("ordinal = %d, want %d", ordinal, tc.want)
			}
		})
	}
}

func TestResolve_Bullet_NoOrdinal(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/><w:lvlText w:val="•"/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`)

	def, ordinal, ok := resolver.Resolve(1, 0)
	if !ok {
		t.Fatal("Resolve() ok = false, want true")
	}
	if def.Format != "bullet" {
		t.Fatalf("Format = %q, want bullet", def.Format)
	}
	if ordinal != 1 {
		t.Fatalf("ordinal = %d, want 1", ordinal)
	}
}

func TestResolve_None_EmptyLevelText(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:numFmt w:val="none"/><w:lvlText w:val="%1."/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`)

	def, _, ok := resolver.Resolve(1, 0)
	if !ok {
		t.Fatal("Resolve() ok = false, want true")
	}
	if def.Format != "none" {
		t.Fatalf("Format = %q, want none", def.Format)
	}
	if def.LevelText != "" {
		t.Fatalf("LevelText = %q, want empty", def.LevelText)
	}
}

func TestResolve_NestedRestart(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, nestedNumberingXML(`<w:lvl w:ilvl="1"><w:numFmt w:val="lowerLetter"/><w:lvlText w:val="%2."/></w:lvl>`))

	for _, tc := range []struct {
		name  string
		level int
		want  int
	}{
		{name: "level0 first", level: 0, want: 1},
		{name: "level1 first", level: 1, want: 1},
		{name: "level1 second", level: 1, want: 2},
		{name: "level0 second", level: 0, want: 2},
		{name: "level1 restarted", level: 1, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ordinal, ok := resolver.Resolve(1, tc.level)
			if !ok {
				t.Fatal("Resolve() ok = false, want true")
			}
			if ordinal != tc.want {
				t.Fatalf("ordinal = %d, want %d", ordinal, tc.want)
			}
		})
	}
}

func TestResolve_NeverRestart(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, nestedNumberingXML(`<w:lvl w:ilvl="1"><w:numFmt w:val="lowerLetter"/><w:lvlText w:val="%2."/><w:lvlRestart w:val="-1"/></w:lvl>`))

	for _, tc := range []struct {
		name  string
		level int
		want  int
	}{
		{name: "level0 first", level: 0, want: 1},
		{name: "level1 first", level: 1, want: 1},
		{name: "level1 second", level: 1, want: 2},
		{name: "level0 second", level: 0, want: 2},
		{name: "level1 continued", level: 1, want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ordinal, ok := resolver.Resolve(1, tc.level)
			if !ok {
				t.Fatal("Resolve() ok = false, want true")
			}
			if ordinal != tc.want {
				t.Fatalf("ordinal = %d, want %d", ordinal, tc.want)
			}
		})
	}
}

func TestResolve_StartOverride(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>
</w:abstractNum>
<w:num w:numId="1">
  <w:abstractNumId w:val="0"/>
  <w:lvlOverride w:ilvl="0"><w:startOverride w:val="3"/></w:lvlOverride>
</w:num>`)

	for _, tc := range []struct {
		name string
		want int
	}{
		{name: "start override", want: 3},
		{name: "next", want: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ordinal, ok := resolver.Resolve(1, 0)
			if !ok {
				t.Fatal("Resolve() ok = false, want true")
			}
			if ordinal != tc.want {
				t.Fatalf("ordinal = %d, want %d", ordinal, tc.want)
			}
		})
	}
}

func TestResolve_LvlOverride_FullReplace(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>
</w:abstractNum>
<w:num w:numId="1">
  <w:abstractNumId w:val="0"/>
  <w:lvlOverride w:ilvl="0">
    <w:lvl w:ilvl="0"><w:start w:val="4"/><w:numFmt w:val="upperRoman"/><w:lvlText w:val="%1."/></w:lvl>
  </w:lvlOverride>
</w:num>`)

	def, ordinal, ok := resolver.Resolve(1, 0)
	if !ok {
		t.Fatal("Resolve() ok = false, want true")
	}
	if def.Format != "upperRoman" {
		t.Fatalf("Format = %q, want upperRoman", def.Format)
	}
	if ordinal != 4 {
		t.Fatalf("ordinal = %d, want 4", ordinal)
	}
}

func TestResolve_UnknownNumID(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:numFmt w:val="decimal"/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`)

	if _, _, ok := resolver.Resolve(99, 0); ok {
		t.Fatal("Resolve() ok = true, want false")
	}
}

func TestResolve_NumIDZero(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:numFmt w:val="decimal"/></w:lvl>
</w:abstractNum>
<w:num w:numId="0"><w:abstractNumId w:val="0"/></w:num>`)

	if _, _, ok := resolver.Resolve(0, 0); ok {
		t.Fatal("Resolve() ok = true, want false")
	}
}

func TestParse_MissingFile(t *testing.T) {
	t.Parallel()

	resolver, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse(nil) error = %v", err)
	}
	if _, _, ok := resolver.Resolve(1, 0); ok {
		t.Fatal("Resolve() ok = true, want false")
	}
}

func TestReset_ClearsCounters(t *testing.T) {
	t.Parallel()

	resolver := parseResolver(t, `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:start w:val="2"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`)

	for _, want := range []int{2, 3} {
		_, ordinal, ok := resolver.Resolve(1, 0)
		if !ok {
			t.Fatal("Resolve() ok = false, want true")
		}
		if ordinal != want {
			t.Fatalf("ordinal = %d, want %d", ordinal, want)
		}
	}

	resolver.Reset()
	_, ordinal, ok := resolver.Resolve(1, 0)
	if !ok {
		t.Fatal("Resolve() ok = false after Reset, want true")
	}
	if ordinal != 2 {
		t.Fatalf("ordinal after Reset = %d, want 2", ordinal)
	}
}

func parseResolver(t *testing.T, innerXML string) *Resolver {
	t.Helper()

	resolver, err := Parse(strings.NewReader(`<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + innerXML + `</w:numbering>`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return resolver
}

func nestedNumberingXML(levelOneXML string) string {
	return `
<w:abstractNum w:abstractNumId="0">
  <w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>
  ` + levelOneXML + `
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`
}
