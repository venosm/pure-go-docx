package numbering

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// LevelDef describes one numbering level definition from word/numbering.xml.
type LevelDef struct {
	Format    string
	LevelText string
	Start     int
	Restart   int
}

// Resolver resolves DOCX numbering definitions and tracks list counters.
//
// Resolve mutates counter state and is intended for single-threaded document
// walks. Callers must not use the same Resolver concurrently from multiple
// goroutines.
type Resolver struct {
	numToAbstract map[int]int
	abstractDefs  map[int]map[int]LevelDef
	overrides     map[int]map[int]LevelDef
	startOverride map[int]map[int]int

	counters map[int]map[int]int
	seen     map[int]map[int]bool
}

// Parse reads word/numbering.xml into a Resolver.
//
// A nil reader returns an empty Resolver and no error.
func Parse(r io.Reader) (*Resolver, error) {
	resolver := newResolver()
	if r == nil {
		return resolver, nil
	}

	p := &parser{
		dec:      xml.NewDecoder(r),
		resolver: resolver,
	}
	p.dec.Strict = false
	p.dec.Entity = xml.HTMLEntity

	for {
		tok, err := p.dec.Token()
		if errors.Is(err, io.EOF) {
			return resolver, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading numbering token: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "abstractNum":
			if err := p.parseAbstractNum(se); err != nil {
				return nil, err
			}
		case "num":
			if err := p.parseNum(se); err != nil {
				return nil, err
			}
		}
	}
}

// Resolve advances the counter for numID and level.
//
// It returns ok=false when the paragraph is not a list item, or when the
// requested numID/level is unknown. Resolve is not safe for concurrent use.
func (r *Resolver) Resolve(numID, level int) (def LevelDef, ordinal int, ok bool) {
	if r == nil || numID == 0 || level < 0 {
		return LevelDef{}, 0, false
	}

	def, ok = r.levelDef(numID, level)
	if !ok {
		return LevelDef{}, 0, false
	}
	if def.Start == 0 {
		def.Start = 1
	}

	r.ensureCounterMaps(numID)
	if !r.seen[numID][level] {
		start := def.Start
		if startOverride, ok := r.startOverrideFor(numID, level); ok {
			start = startOverride
		}
		if start == 0 {
			start = 1
		}
		r.counters[numID][level] = start
		r.seen[numID][level] = true
	}

	ordinal = r.counters[numID][level]
	r.counters[numID][level] = ordinal + 1
	r.resetDeeperLevels(numID, level)

	return def, ordinal, true
}

// Reset clears all mutable counters while preserving parsed numbering definitions.
func (r *Resolver) Reset() {
	if r == nil {
		return
	}
	r.counters = make(map[int]map[int]int)
	r.seen = make(map[int]map[int]bool)
}

type parser struct {
	dec      *xml.Decoder
	resolver *Resolver
}

type levelOverride struct {
	level    int
	def      LevelDef
	hasDef   bool
	start    int
	hasStart bool
}

func newResolver() *Resolver {
	return &Resolver{
		numToAbstract: make(map[int]int),
		abstractDefs:  make(map[int]map[int]LevelDef),
		overrides:     make(map[int]map[int]LevelDef),
		startOverride: make(map[int]map[int]int),
		counters:      make(map[int]map[int]int),
		seen:          make(map[int]map[int]bool),
	}
}

func (p *parser) parseAbstractNum(start xml.StartElement) error {
	abstractNumID := parseInt(attr(start, "abstractNumId"), -1)
	if abstractNumID < 0 {
		return p.skipElement(start)
	}
	levels := ensureLevelMap(p.resolver.abstractDefs, abstractNumID)

	for {
		tok, err := p.dec.Token()
		if err != nil {
			return fmt.Errorf("parsing abstract numbering %d: %w", abstractNumID, err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "lvl" {
				level, def, err := p.parseLevel(t)
				if err != nil {
					return err
				}
				if level >= 0 {
					levels[level] = def
				}
				continue
			}
			if err := p.skipElement(t); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name.Local == "abstractNum" {
				return nil
			}
		}
	}
}

func (p *parser) parseNum(start xml.StartElement) error {
	numID := parseInt(attr(start, "numId"), -1)
	if numID < 0 {
		return p.skipElement(start)
	}

	for {
		tok, err := p.dec.Token()
		if err != nil {
			return fmt.Errorf("parsing numbering instance %d: %w", numID, err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "abstractNumId":
				abstractNumID := parseInt(attr(t, "val"), -1)
				if abstractNumID >= 0 {
					p.resolver.numToAbstract[numID] = abstractNumID
				}
			case "lvlOverride":
				override, err := p.parseLevelOverride(t)
				if err != nil {
					return err
				}
				if override.hasDef {
					ensureLevelMap(p.resolver.overrides, numID)[override.level] = override.def
				}
				if override.hasStart {
					ensureLevelMap(p.resolver.startOverride, numID)[override.level] = override.start
				}
			default:
				if err := p.skipElement(t); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "num" {
				return nil
			}
		}
	}
}

func (p *parser) parseLevelOverride(start xml.StartElement) (levelOverride, error) {
	override := levelOverride{
		level: parseInt(attr(start, "ilvl"), -1),
	}
	if override.level < 0 {
		return override, p.skipElement(start)
	}

	for {
		tok, err := p.dec.Token()
		if err != nil {
			return levelOverride{}, fmt.Errorf("parsing level override %d: %w", override.level, err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "startOverride":
				override.start = parseInt(attr(t, "val"), 1)
				override.hasStart = true
			case "lvl":
				level, def, err := p.parseLevel(t)
				if err != nil {
					return levelOverride{}, err
				}
				if level >= 0 {
					override.level = level
				}
				override.def = def
				override.hasDef = true
			default:
				if err := p.skipElement(t); err != nil {
					return levelOverride{}, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "lvlOverride" {
				return override, nil
			}
		}
	}
}

func (p *parser) parseLevel(start xml.StartElement) (int, LevelDef, error) {
	level := parseInt(attr(start, "ilvl"), -1)
	def := LevelDef{
		Format: "decimal",
		Start:  1,
	}

	for {
		tok, err := p.dec.Token()
		if err != nil {
			return 0, LevelDef{}, fmt.Errorf("parsing level %d: %w", level, err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "start":
				def.Start = parseInt(attr(t, "val"), 1)
			case "numFmt":
				if format := attr(t, "val"); format != "" {
					def.Format = format
				}
			case "lvlText":
				def.LevelText = attr(t, "val")
			case "lvlRestart":
				def.Restart = parseInt(attr(t, "val"), 0)
			default:
				if err := p.skipElement(t); err != nil {
					return 0, LevelDef{}, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "lvl" {
				if def.Format == "none" {
					def.LevelText = ""
				}
				return level, def, nil
			}
		}
	}
}

func (p *parser) skipElement(start xml.StartElement) error {
	depth := 1
	for depth > 0 {
		tok, err := p.dec.Token()
		if err != nil {
			return fmt.Errorf("skipping %s: %w", start.Name.Local, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == start.Name.Local {
				depth++
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				depth--
			}
		}
	}
	return nil
}

func (r *Resolver) levelDef(numID, level int) (LevelDef, bool) {
	if levels, ok := r.overrides[numID]; ok {
		if def, ok := levels[level]; ok {
			return def, true
		}
	}
	abstractNumID, ok := r.numToAbstract[numID]
	if !ok {
		return LevelDef{}, false
	}
	levels, ok := r.abstractDefs[abstractNumID]
	if !ok {
		return LevelDef{}, false
	}
	def, ok := levels[level]
	return def, ok
}

func (r *Resolver) startOverrideFor(numID, level int) (int, bool) {
	levels, ok := r.startOverride[numID]
	if !ok {
		return 0, false
	}
	start, ok := levels[level]
	return start, ok
}

func (r *Resolver) resetDeeperLevels(numID, level int) {
	for deeper := range r.seen[numID] {
		if deeper <= level {
			continue
		}
		def, ok := r.levelDef(numID, deeper)
		if !ok || def.Restart == -1 {
			continue
		}
		if def.Restart == 0 || def.Restart == level {
			delete(r.counters[numID], deeper)
			delete(r.seen[numID], deeper)
		}
	}
}

func (r *Resolver) ensureCounterMaps(numID int) {
	if _, ok := r.counters[numID]; !ok {
		r.counters[numID] = make(map[int]int)
	}
	if _, ok := r.seen[numID]; !ok {
		r.seen[numID] = make(map[int]bool)
	}
}

func ensureLevelMap[T int | LevelDef](maps map[int]map[int]T, id int) map[int]T {
	levels, ok := maps[id]
	if !ok {
		levels = make(map[int]T)
		maps[id] = levels
	}
	return levels
}

func attr(se xml.StartElement, local string) string {
	for _, attr := range se.Attr {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
