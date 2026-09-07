package body

import (
	"encoding/xml"
	"strconv"
)

type drawingState struct {
	relID     string
	altText   string
	widthEMU  int64
	heightEMU int64
}

func (p *parser) parseDrawing() (*ImageRef, error) {
	state := drawingState{}

	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "AlternateContent":
				if err := p.parseAlternateContentDrawing(&state); err != nil {
					return nil, err
				}
			default:
				state.applyStart(t)
			}
		case xml.EndElement:
			if t.Name.Local == "drawing" {
				if state.relID == "" {
					return nil, nil
				}
				image := &ImageRef{
					ID:        imageID(p.sourcePart, state.relID),
					PartName:  p.sourcePart,
					RelID:     state.relID,
					AltText:   state.altText,
					WidthEMU:  state.widthEMU,
					HeightEMU: state.heightEMU,
				}
				if rel, ok := p.relationships[state.relID]; ok {
					image.Filename = rel.Target
				}
				return image, nil
			}
		}
	}
}

func (p *parser) parseAlternateContentDrawing(state *drawingState) error {
	chosen := false
	for {
		tok, err := p.next()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "Choice":
				if chosen {
					if err := p.skipElement(t); err != nil {
						return err
					}
					continue
				}
				if err := p.parseDrawingUntil("Choice", state); err != nil {
					return err
				}
				chosen = true
			case "Fallback":
				if chosen {
					if err := p.skipElement(t); err != nil {
						return err
					}
					continue
				}
				if err := p.parseDrawingUntil("Fallback", state); err != nil {
					return err
				}
				chosen = true
			default:
				if err := p.skipElement(t); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "AlternateContent" {
				return nil
			}
		}
	}
}

func (p *parser) parseDrawingUntil(endLocal string, state *drawingState) error {
	for {
		tok, err := p.next()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "AlternateContent" {
				if err := p.parseAlternateContentDrawing(state); err != nil {
					return err
				}
				continue
			}
			state.applyStart(t)
		case xml.EndElement:
			if t.Name.Local == endLocal {
				return nil
			}
		}
	}
}

func (s *drawingState) applyStart(start xml.StartElement) {
	switch start.Name.Local {
	case "docPr":
		if descr := attr(start, "descr"); descr != "" {
			s.altText = descr
		}
	case "extent":
		if s.widthEMU == 0 {
			s.widthEMU = parseInt64(attr(start, "cx"))
			s.heightEMU = parseInt64(attr(start, "cy"))
		}
	case "blip":
		if embed := attr(start, "embed"); embed != "" {
			s.relID = embed
		}
	}
}

func imageID(partName, relID string) string {
	if partName == "" {
		return relID
	}
	return partName + "#" + relID
}

func parseInt64(value string) int64 {
	if value == "" {
		return 0
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
