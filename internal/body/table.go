package body

import "encoding/xml"

type rawCell struct {
	blocks []Block
	hSpan  int
	vMerge MergeKind
}

type rawRow struct {
	cells []rawCell
}

type rawTable struct {
	gridWidth int
	rows      []rawRow
}

type denseTable struct {
	grid   [][]Cell
	starts [][]bool
}

func (p *parser) parseTable() (*Table, error) {
	raw := rawTable{}
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tblGrid":
				width, err := p.parseTableGrid()
				if err != nil {
					return nil, err
				}
				raw.gridWidth = width
			case "tr":
				row, err := p.parseRawTableRow()
				if err != nil {
					return nil, err
				}
				raw.rows = append(raw.rows, row)
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "tbl" {
				return &Table{Grid: densifyTable(raw)}, nil
			}
		}
	}
}

func (p *parser) parseTableGrid() (int, error) {
	width := 0
	for {
		tok, err := p.next()
		if err != nil {
			return 0, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "gridCol" {
				width++
			}
			if err := p.skipElement(t); err != nil {
				return 0, err
			}
		case xml.EndElement:
			if t.Name.Local == "tblGrid" {
				return width, nil
			}
		}
	}
}

func (p *parser) parseRawTableRow() (rawRow, error) {
	row := rawRow{}
	for {
		tok, err := p.next()
		if err != nil {
			return rawRow{}, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "tc" {
				cell, err := p.parseRawTableCell()
				if err != nil {
					return rawRow{}, err
				}
				row.cells = append(row.cells, cell)
				continue
			}
			if err := p.skipElement(t); err != nil {
				return rawRow{}, err
			}
		case xml.EndElement:
			if t.Name.Local == "tr" {
				return row, nil
			}
		}
	}
}

func (p *parser) parseRawTableCell() (rawCell, error) {
	cell := rawCell{hSpan: 1}
	for {
		tok, err := p.next()
		if err != nil {
			return rawCell{}, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tcPr":
				if err := p.parseTableCellProperties(&cell); err != nil {
					return rawCell{}, err
				}
			case "p":
				paragraph, err := p.parseParagraph()
				if err != nil {
					return rawCell{}, err
				}
				cell.blocks = append(cell.blocks, paragraph)
			case "tbl":
				table, err := p.parseTable()
				if err != nil {
					return rawCell{}, err
				}
				cell.blocks = append(cell.blocks, table)
			case "sdt":
				blocks, err := p.parseSDT()
				if err != nil {
					return rawCell{}, err
				}
				cell.blocks = append(cell.blocks, blocks...)
			default:
				if err := p.skipElement(t); err != nil {
					return rawCell{}, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "tc" {
				if cell.hSpan < 1 {
					cell.hSpan = 1
				}
				return cell, nil
			}
		}
	}
}

func (p *parser) parseTableCellProperties(cell *rawCell) error {
	for {
		tok, err := p.next()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "gridSpan":
				cell.hSpan = xmlInt(attr(t, "val"), 1)
				if cell.hSpan < 1 {
					cell.hSpan = 1
				}
			case "vMerge":
				switch attr(t, "val") {
				case "restart":
					cell.vMerge = MergeRestart
				default:
					cell.vMerge = MergeContinue
				}
			default:
				if err := p.skipElement(t); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "tcPr" {
				return nil
			}
		}
	}
}

func densifyTable(raw rawTable) [][]Cell {
	width := raw.gridWidth
	if computed := computedGridWidth(raw.rows); computed > width {
		width = computed
	}
	if width <= 0 {
		return nil
	}

	dense := denseTable{
		grid:   make([][]Cell, 0, len(raw.rows)),
		starts: make([][]bool, 0, len(raw.rows)),
	}

	for _, rawRow := range raw.rows {
		row := make([]Cell, 0, width)
		starts := make([]bool, 0, width)
		col := 0
		for _, rawCell := range rawRow.cells {
			span := rawCell.hSpan
			if span < 1 {
				span = 1
			}
			if col >= width {
				break
			}
			row = append(row, Cell{
				Blocks: rawCell.blocks,
				HSpan:  span,
				VMerge: rawCell.vMerge,
			})
			starts = append(starts, true)
			col++
			for i := 1; i < span && col < width; i++ {
				row = append(row, Cell{HSpan: 1})
				starts = append(starts, false)
				col++
			}
		}
		for col < width {
			row = append(row, Cell{HSpan: 1})
			starts = append(starts, false)
			col++
		}
		dense.grid = append(dense.grid, row)
		dense.starts = append(dense.starts, starts)
	}

	inheritVerticalMerges(&dense)
	return dense.grid
}

func computedGridWidth(rows []rawRow) int {
	maxWidth := 0
	for _, row := range rows {
		width := 0
		for _, cell := range row.cells {
			span := cell.hSpan
			if span < 1 {
				span = 1
			}
			width += span
		}
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

func inheritVerticalMerges(table *denseTable) {
	for rowIndex := range table.grid {
		for colIndex := range table.grid[rowIndex] {
			if table.grid[rowIndex][colIndex].VMerge != MergeContinue {
				continue
			}
			origin, ok := table.findVerticalMergeOrigin(rowIndex, colIndex)
			if !ok {
				// TODO: orphan vMerge logging.
				continue
			}
			table.grid[rowIndex][colIndex].Blocks = origin.Blocks
			table.grid[rowIndex][colIndex].HSpan = origin.HSpan
		}
	}
}

func (t *denseTable) findVerticalMergeOrigin(rowIndex, colIndex int) (Cell, bool) {
	for r := rowIndex - 1; r >= 0; r-- {
		cell, ok := t.cellCoveringColumn(r, colIndex)
		if !ok {
			continue
		}
		if cell.VMerge == MergeContinue {
			continue
		}
		return cell, true
	}
	return Cell{}, false
}

func (t *denseTable) cellCoveringColumn(rowIndex, colIndex int) (Cell, bool) {
	if rowIndex < 0 || rowIndex >= len(t.grid) {
		return Cell{}, false
	}
	row := t.grid[rowIndex]
	starts := t.starts[rowIndex]
	for start := colIndex; start >= 0; start-- {
		if start >= len(row) || start >= len(starts) || !starts[start] {
			continue
		}
		span := row[start].HSpan
		if span < 1 {
			span = 1
		}
		if start+span > colIndex {
			return row[start], true
		}
	}
	return Cell{}, false
}
