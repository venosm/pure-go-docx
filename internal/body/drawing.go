package body

import (
	"encoding/xml"
	"strconv"
)

func (p *parser) parseDrawing() (*ImageRef, error) {
	var relID string
	var altText string
	var widthEMU int64
	var heightEMU int64

	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "docPr":
				if descr := attr(t, "descr"); descr != "" {
					altText = descr
				}
			case "extent":
				if widthEMU == 0 {
					widthEMU = parseInt64(attr(t, "cx"))
					heightEMU = parseInt64(attr(t, "cy"))
				}
			case "blip":
				if embed := attr(t, "embed"); embed != "" {
					relID = embed
				}
			}
		case xml.EndElement:
			if t.Name.Local == "drawing" {
				if relID == "" {
					return nil, nil
				}
				image := &ImageRef{
					RelID:     relID,
					AltText:   altText,
					WidthEMU:  widthEMU,
					HeightEMU: heightEMU,
				}
				if rel, ok := p.relationships[relID]; ok {
					image.Filename = rel.Target
				}
				return image, nil
			}
		}
	}
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
