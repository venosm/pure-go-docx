package godocx

import (
	"bytes"
	"testing"

	"github.com/venosm/pure-go-docx/internal/testutil"
)

func TestTable_FlatGrid_2x2(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>`+tc("A")+tc("B")+`</w:tr>
  <w:tr>`+tc("C")+tc("D")+`</w:tr>
</w:tbl>`)

	if got, want := len(table.Grid), 2; got != want {
		t.Fatalf("rows = %d, want %d", got, want)
	}
	if got, want := len(table.Grid[0]), 2; got != want {
		t.Fatalf("cols = %d, want %d", got, want)
	}
	if got, want := cellText(table.Grid[1][1]), "D"; got != want {
		t.Fatalf("cell text = %q, want %q", got, want)
	}
}

func TestTable_HorizontalMerge_GridSpan(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>
    <w:tc><w:tcPr><w:gridSpan w:val="3"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>
    `+tc("B")+`
  </w:tr>
</w:tbl>`)

	if got, want := len(table.Grid[0]), 4; got != want {
		t.Fatalf("cols = %d, want %d", got, want)
	}
	if got, want := table.Grid[0][0].HSpan, 3; got != want {
		t.Fatalf("HSpan = %d, want %d", got, want)
	}
	if got, want := cellText(table.Grid[0][0]), "A"; got != want {
		t.Fatalf("cell 0 text = %q, want %q", got, want)
	}
	if got := cellText(table.Grid[0][1]); got != "" {
		t.Fatalf("covered cell text = %q, want empty", got)
	}
	if got, want := table.Grid[0][3].HSpan, 1; got != want {
		t.Fatalf("cell 3 HSpan = %d, want %d", got, want)
	}
	if got, want := cellText(table.Grid[0][3]), "B"; got != want {
		t.Fatalf("cell 3 text = %q, want %q", got, want)
	}
}

func TestTable_VerticalMerge_Inheritance(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>
    <w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>Smlouva</w:t></w:r></w:p></w:tc>
    `+tc("42")+`
  </w:tr>
  <w:tr>
    <w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc>
    `+tc("43")+`
  </w:tr>
  <w:tr>
    <w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc>
    `+tc("44")+`
  </w:tr>
</w:tbl>`)

	for _, row := range []int{1, 2} {
		if got, want := table.Grid[row][0].VMerge, MergeContinue; got != want {
			t.Fatalf("row %d VMerge = %d, want %d", row, got, want)
		}
		if got, want := cellText(table.Grid[row][0]), "Smlouva"; got != want {
			t.Fatalf("row %d inherited text = %q, want %q", row, got, want)
		}
	}
}

func TestTable_VerticalMerge_RestartMidTable(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/></w:tblGrid>
  <w:tr><w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc></w:tr>
  <w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc></w:tr>
  <w:tr><w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc></w:tr>
  <w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc></w:tr>
</w:tbl>`)

	if got, want := cellText(table.Grid[1][0]), "A"; got != want {
		t.Fatalf("first continuation = %q, want %q", got, want)
	}
	if got, want := cellText(table.Grid[3][0]), "B"; got != want {
		t.Fatalf("second continuation = %q, want %q", got, want)
	}
}

func TestTable_CombinedMerge(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>
    <w:tc><w:tcPr><w:gridSpan w:val="2"/><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>
  </w:tr>
  <w:tr>
    <w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc>
  </w:tr>
</w:tbl>`)

	if got, want := len(table.Grid[1]), 2; got != want {
		t.Fatalf("cols = %d, want %d", got, want)
	}
	if got, want := table.Grid[1][0].HSpan, 2; got != want {
		t.Fatalf("continuation HSpan = %d, want %d", got, want)
	}
	if got, want := cellText(table.Grid[1][0]), "A"; got != want {
		t.Fatalf("continuation text = %q, want %q", got, want)
	}
	if got := cellText(table.Grid[1][1]); got != "" {
		t.Fatalf("covered continuation cell text = %q, want empty", got)
	}
}

func TestTable_NestedTable_WithMerges(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>
    `+tc("outer")+`
    <w:tc>
      <w:tbl>
        <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
        <w:tr>
          <w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>inner</w:t></w:r></w:p></w:tc>
          `+tc("x")+`
        </w:tr>
        <w:tr>
          <w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc>
          `+tc("y")+`
        </w:tr>
      </w:tbl>
    </w:tc>
  </w:tr>
  <w:tr>`+tc("left")+tc("right")+`</w:tr>
</w:tbl>`)

	nested, ok := table.Grid[0][1].Blocks[0].(*Table)
	if !ok {
		t.Fatalf("nested block type = %T, want *Table", table.Grid[0][1].Blocks[0])
	}
	if got, want := cellText(nested.Grid[1][0]), "inner"; got != want {
		t.Fatalf("nested inherited text = %q, want %q", got, want)
	}
}

func TestTable_PadShortRow(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>`+tc("A")+tc("B")+tc("C")+`</w:tr>
  <w:tr>`+tc("D")+`</w:tr>
</w:tbl>`)

	if got, want := len(table.Grid[1]), 3; got != want {
		t.Fatalf("cols = %d, want %d", got, want)
	}
	if got := cellText(table.Grid[1][1]); got != "" {
		t.Fatalf("padded cell text = %q, want empty", got)
	}
}

func TestTable_NoGridDefined(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tr>
    <w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>
    `+tc("B")+`
  </w:tr>
  <w:tr>`+tc("C")+tc("D")+tc("E")+`</w:tr>
</w:tbl>`)

	if got, want := len(table.Grid[0]), 3; got != want {
		t.Fatalf("cols = %d, want %d", got, want)
	}
	if got, want := cellText(table.Grid[0][2]), "B"; got != want {
		t.Fatalf("cell text = %q, want %q", got, want)
	}
}

func TestTable_OrphanContinuation(t *testing.T) {
	t.Parallel()

	table := openSingleTable(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/></w:tblGrid>
  <w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc></w:tr>
</w:tbl>`)

	if got, want := table.Grid[0][0].VMerge, MergeContinue; got != want {
		t.Fatalf("VMerge = %d, want %d", got, want)
	}
	if got := cellText(table.Grid[0][0]); got != "" {
		t.Fatalf("orphan continuation text = %q, want empty", got)
	}
}

func openSingleTable(t *testing.T, tableXML string) *Table {
	t.Helper()

	data := testutil.BuildDocx(t, tableXML, "")
	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	if len(doc.Body) != 1 {
		t.Fatalf("len(Body) = %d, want 1", len(doc.Body))
	}
	table, ok := doc.Body[0].(*Table)
	if !ok {
		t.Fatalf("Body[0] type = %T, want *Table", doc.Body[0])
	}
	return table
}

func tc(text string) string {
	return `<w:tc><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:tc>`
}
