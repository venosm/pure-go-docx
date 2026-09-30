package main

import (
	"fmt"
	"strings"
)

// fixture is one generated DOCX file written into the testdata directory.
type fixture struct {
	// name is the file name inside the testdata directory.
	name string
	// description explains which parser features the fixture covers.
	description string
	// build produces the complete archive content.
	build func() ([]byte, error)
}

// fixtures lists every generated fixture in a stable order.
func fixtures() []fixture {
	return []fixture{
		{
			name:        "simple.docx",
			description: "Paragraphs, headings, tabs, line breaks, run formatting, and hyperlinks.",
			build:       buildSimple,
		},
		{
			name:        "tables.docx",
			description: "Flat tables with a header row and multiple data rows.",
			build:       buildTables,
		},
		{
			name:        "tables-merged.docx",
			description: "Tables using gridSpan and vMerge so merges must be resolved.",
			build:       buildMergedTables,
		},
		{
			name:        "nested-tables.docx",
			description: "Recursive tables placed inside table cells.",
			build:       buildNestedTables,
		},
		{
			name:        "lists-nested.docx",
			description: "Numbered and bulleted lists with nesting and a numbering override.",
			build:       buildNestedLists,
		},
		{
			name:        "images.docx",
			description: "Embedded PNG, JPEG, GIF, and SVG images with English alt text.",
			build:       buildImages,
		},
		{
			name:        "unicode-whitespace.docx",
			description: "English text with international characters and preserved run whitespace.",
			build:       buildUnicodeWhitespace,
		},
		{
			name:        "headers-footers.docx",
			description: "Two headers, one footer, and a header-scoped image relationship.",
			build:       buildHeadersFooters,
		},
		{
			name:        "footnotes-endnotes.docx",
			description: "Footnote and endnote definitions with in-body references.",
			build:       buildNotes,
		},
		{
			name:        "sdt-content-controls.docx",
			description: "Structured document tags wrapping body blocks and table cells.",
			build:       buildStructuredTags,
		},
		{
			name:        "fields-alternate-content.docx",
			description: "Complex fields, fldSimple, and mc:AlternateContent choice/fallback markup.",
			build:       buildFieldsAlternateContent,
		},
		{
			name:        "service-agreement-en.docx",
			description: "Full English service agreement combining every supported feature.",
			build:       buildServiceAgreement,
		},
		{
			name:        "malformed-truncated.docx",
			description: "Truncated archive without a central directory, for robustness tests.",
			build:       buildTruncated,
		},
	}
}

func buildSimple() ([]byte, error) {
	body := strings.Join([]string{
		heading(1, "Quarterly Service Report"),
		paragraph(text("This fixture covers ordinary paragraphs, headings, tabs, line breaks, and basic run formatting.")),
		paragraph(
			styledText("Bold text", true, false, false),
			text(", "),
			styledText("italic text", false, true, false),
			text(", and "),
			styledText("underlined text", false, false, true),
			text(" appear in one paragraph."),
		),
		paragraph(text("Region"), tabRun(), text("Contact"), tabRun(), text("Status")),
		paragraph(text("First line of the address."), breakRun(), text("Second line after an explicit break.")),
		heading(2, "Scope of Work"),
		paragraph(text("The supplier provides onboarding, configuration, and training services for the reporting platform.")),
		heading(3, "Deliverables"),
		paragraph(text("A signed acceptance protocol closes every delivery milestone.")),
		paragraph(
			text("See the "),
			hyperlink("rId1", styledText("project documentation", false, false, true)),
			text(" for the full specification."),
		),
		paragraph(text("Jump to the "), anchorLink("appendix", text("appendix")), text(".")),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	a.addRelationships("word/document.xml", []relationship{
		{id: "rId1", relType: relTypeHyperlink, target: "https://example.com/docs/reporting-platform", targetMode: "External"},
	})
	return a.bytes()
}

func buildTables() ([]byte, error) {
	body := strings.Join([]string{
		heading(1, "Price List"),
		paragraph(text("The table below lists the ordered items and their unit prices.")),
		table(4, [][]cell{
			{boldCell("Item"), boldCell("Quantity"), boldCell("Unit price"), boldCell("Total")},
			{textCell("Annual licence"), textCell("12"), textCell("EUR 240.00"), textCell("EUR 2,880.00")},
			{textCell("Implementation"), textCell("1"), textCell("EUR 4,500.00"), textCell("EUR 4,500.00")},
			{textCell("Support hours"), textCell("40"), textCell("EUR 75.00"), textCell("EUR 3,000.00")},
		}),
		paragraph(text("All prices exclude value added tax.")),
		table(2, [][]cell{
			{boldCell("Contact role"), boldCell("Name")},
			{textCell("Contract manager"), textCell("Jordan Ellis")},
			{textCell("Technical lead"), textCell("Priya Raman")},
		}),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	return a.bytes()
}

func buildMergedTables() ([]byte, error) {
	body := strings.Join([]string{
		heading(1, "Delivery Schedule"),
		paragraph(text("The title row spans three columns and the phase column is merged vertically.")),
		table(3, [][]cell{
			{{blocks: paragraph(styledText("Implementation phases", true, false, false)), gridSpan: 3}},
			{boldCell("Phase"), boldCell("Milestone"), boldCell("Due date")},
			{
				{blocks: paragraph(text("Phase 1")), vMerge: "restart"},
				textCell("Kick-off workshop"),
				textCell("2026-01-15"),
			},
			{
				{vMerge: "continue"},
				textCell("Environment handover"),
				textCell("2026-02-02"),
			},
			{
				{blocks: paragraph(text("Phase 2")), vMerge: "restart"},
				textCell("Data migration"),
				textCell("2026-03-10"),
			},
			{
				{vMerge: "continue"},
				textCell("Acceptance testing"),
				textCell("2026-03-28"),
			},
			{
				{blocks: paragraph(text("Total duration: 11 weeks")), gridSpan: 2},
				textCell("Confirmed"),
			},
		}),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	return a.bytes()
}

func buildNestedTables() ([]byte, error) {
	inner := table(2, [][]cell{
		{boldCell("Sub-item"), boldCell("Amount")},
		{textCell("Setup fee"), textCell("EUR 1,200.00")},
		{textCell("Training day"), textCell("EUR 800.00")},
	})

	body := strings.Join([]string{
		heading(1, "Cost Breakdown"),
		table(2, [][]cell{
			{boldCell("Category"), boldCell("Detail")},
			{
				textCell("One-off charges"),
				{blocks: paragraph(text("Broken down in the nested table:")) + inner},
			},
			{
				textCell("Recurring charges"),
				{blocks: paragraph(text("Invoiced monthly in arrears."))},
			},
		}),
		paragraph(text("The nested table is parsed as a block inside its parent cell.")),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	return a.bytes()
}

func buildNestedLists() ([]byte, error) {
	body := strings.Join([]string{
		heading(1, "Acceptance Checklist"),
		paragraph(text("Numbered list with three nesting levels:")),
		listParagraph(1, 0, text("Prepare the test environment.")),
		listParagraph(1, 1, text("Restore the anonymised data set.")),
		listParagraph(1, 1, text("Verify the service accounts.")),
		listParagraph(1, 2, text("Confirm read-only access for auditors.")),
		listParagraph(1, 0, text("Run the acceptance test suite.")),
		listParagraph(1, 1, text("Record every deviation in the defect log.")),
		listParagraph(1, 0, text("Sign the acceptance protocol.")),
		paragraph(text("Bulleted list with two nesting levels:")),
		listParagraph(2, 0, text("Functional requirements")),
		listParagraph(2, 1, text("Search and filtering")),
		listParagraph(2, 1, text("Export to spreadsheet")),
		listParagraph(2, 0, text("Non-functional requirements")),
		listParagraph(2, 1, text("Response time under two seconds")),
		paragraph(text("Numbered list restarted at five through a numbering override:")),
		listParagraph(3, 0, text("Continued item after a restart override.")),
		listParagraph(3, 0, text("Second item after the restart override.")),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	a.addXMLPart("word/numbering.xml", contentTypeNumbering, numberingXML())
	a.addRelationships("word/document.xml", []relationship{
		{id: "rId1", relType: relTypeNumbering, target: "numbering.xml"},
	})
	return a.bytes()
}

// numberingXML defines a decimal/letter/roman list, a bulleted list, and a
// numbering instance that restarts the decimal list at five.
func numberingXML() []byte {
	inner := `<w:abstractNum w:abstractNumId="0">` +
		`<w:multiLevelType w:val="multilevel"/>` +
		`<w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl>` +
		`<w:lvl w:ilvl="1"><w:start w:val="1"/><w:numFmt w:val="lowerLetter"/><w:lvlText w:val="%2)"/></w:lvl>` +
		`<w:lvl w:ilvl="2"><w:start w:val="1"/><w:numFmt w:val="lowerRoman"/><w:lvlText w:val="%3."/></w:lvl>` +
		`</w:abstractNum>` +
		`<w:abstractNum w:abstractNumId="1">` +
		`<w:multiLevelType w:val="hybridMultilevel"/>` +
		`<w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="&#8226;"/></w:lvl>` +
		`<w:lvl w:ilvl="1"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="&#9702;"/></w:lvl>` +
		`</w:abstractNum>` +
		`<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>` +
		`<w:num w:numId="2"><w:abstractNumId w:val="1"/></w:num>` +
		`<w:num w:numId="3"><w:abstractNumId w:val="0"/>` +
		`<w:lvlOverride w:ilvl="0"><w:startOverride w:val="5"/></w:lvlOverride>` +
		`</w:num>`
	return wordPartXML("numbering", inner)
}

func buildImages() ([]byte, error) {
	pngData, err := pngBytes()
	if err != nil {
		return nil, err
	}
	jpegData, err := jpegBytes()
	if err != nil {
		return nil, err
	}
	gifData, err := gifBytes()
	if err != nil {
		return nil, err
	}

	body := strings.Join([]string{
		heading(1, "Embedded Images"),
		paragraph(text("Each paragraph below embeds one image format with alt text.")),
		imageParagraph(1, "rId1", "logo.png", "Company logo in PNG format"),
		imageParagraph(2, "rId2", "photo.jpeg", "Office photograph in JPEG format"),
		imageParagraph(3, "rId3", "icon.gif", "Status icon in GIF format"),
		imageParagraph(4, "rId4", "badge.svg", "Certification badge in SVG format"),
		paragraph(text("An image inside a table cell:")),
		table(2, [][]cell{
			{boldCell("Preview"), boldCell("Description")},
			{
				{blocks: imageParagraph(5, "rId1", "logo.png", "Company logo repeated in a table cell")},
				textCell("The same relationship is referenced twice."),
			},
		}),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	a.addPart("word/media/logo.png", pngData)
	a.addPart("word/media/photo.jpeg", jpegData)
	a.addPart("word/media/icon.gif", gifData)
	a.addPart("word/media/badge.svg", []byte(svgBadge))
	a.addRelationships("word/document.xml", []relationship{
		{id: "rId1", relType: relTypeImage, target: "media/logo.png"},
		{id: "rId2", relType: relTypeImage, target: "media/photo.jpeg"},
		{id: "rId3", relType: relTypeImage, target: "media/icon.gif"},
		{id: "rId4", relType: relTypeImage, target: "media/badge.svg"},
	})
	return a.bytes()
}

func buildUnicodeWhitespace() ([]byte, error) {
	body := strings.Join([]string{
		heading(1, "International Characters and Whitespace"),
		paragraph(text("The supplier is registered as Ångström Systems Ltd.")),
		paragraph(
			text("Leading and trailing spaces "),
			text(" are preserved across "),
			text(" adjacent runs."),
		),
		paragraph(text("Names with diacritics: José Almeida, Zoë Bjärk, François Muller, Andrzej Wójcik.")),
		paragraph(text("Currency and punctuation: EUR 1 250,50 — GBP 980.00 – “quoted term” … 50 %.")),
		paragraph(text("Symbols and scripts: © 2026, µs latency, Ω resistance, 年度报告, مرحبا.")),
		paragraph(text("A non-breaking space keeps the unit together: 12 kW.")),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	return a.bytes()
}

func buildHeadersFooters() ([]byte, error) {
	pngData, err := pngBytes()
	if err != nil {
		return nil, err
	}

	body := strings.Join([]string{
		heading(1, "Document With Headers and Footers"),
		paragraph(text("The default header, the first-page header, and the footer are separate package parts.")),
		paragraph(text("The first-page header embeds a logo through its own relationship part.")),
		sectionProperties(
			headerReference("first", "rId2"),
			headerReference("default", "rId1"),
			footerReference("default", "rId3"),
		),
	}, "")

	defaultHeader := wordPartXML("hdr", strings.Join([]string{
		paragraph(text("Reporting Platform — Service Documentation")),
		paragraph(text("Restricted distribution")),
	}, ""))

	firstHeader := wordPartXML("hdr", strings.Join([]string{
		imageParagraph(10, "rId1", "logo.png", "Supplier logo in the first-page header"),
		paragraph(text("Supplier: Northwind Services Ltd.")),
	}, ""))

	footer := wordPartXML("ftr", strings.Join([]string{
		paragraph(
			text("Page "),
			complexField(" PAGE ", "1"),
			text(" of "),
			complexField(" NUMPAGES ", "3"),
		),
		paragraph(text("Contract reference: NS-2026-0147")),
	}, ""))

	a := newArchive()
	a.addDocument(body)
	a.addXMLPart("word/header1.xml", contentTypeHeader, defaultHeader)
	a.addXMLPart("word/header2.xml", contentTypeHeader, firstHeader)
	a.addXMLPart("word/footer1.xml", contentTypeFooter, footer)
	a.addPart("word/media/logo.png", pngData)
	a.addRelationships("word/document.xml", []relationship{
		{id: "rId1", relType: relTypeHeader, target: "header1.xml"},
		{id: "rId2", relType: relTypeHeader, target: "header2.xml"},
		{id: "rId3", relType: relTypeFooter, target: "footer1.xml"},
	})
	a.addRelationships("word/header2.xml", []relationship{
		{id: "rId1", relType: relTypeImage, target: "media/logo.png"},
	})
	return a.bytes()
}

func buildNotes() ([]byte, error) {
	body := strings.Join([]string{
		heading(1, "Notes and References"),
		paragraph(
			text("The delivery deadline is binding"),
			noteRun("footnote", "2"),
			text(" and the penalty is capped"),
			noteRun("footnote", "3"),
			text("."),
		),
		paragraph(
			text("The governing law is described in the closing section"),
			noteRun("endnote", "2"),
			text("."),
		),
		paragraph(text("A footnote reference may also appear inside a table cell.")),
		table(2, [][]cell{
			{boldCell("Clause"), boldCell("Note")},
			{
				textCell("Warranty"),
				{blocks: paragraph(text("24 months"), noteRun("footnote", "4"))},
			},
		}),
		sectionProperties(),
	}, "")

	footnotes := wordPartXML("footnotes", strings.Join([]string{
		`<w:footnote w:type="separator" w:id="-1"><w:p><w:r><w:separator/></w:r></w:p></w:footnote>`,
		`<w:footnote w:type="continuationSeparator" w:id="0"><w:p><w:r><w:continuationSeparator/></w:r></w:p></w:footnote>`,
		noteDefinition("footnote", "2", "The deadline follows the schedule in Annex 1."),
		noteDefinition("footnote", "3", "The penalty may not exceed ten per cent of the contract price."),
		noteDefinition("footnote", "4", "The warranty period starts on the acceptance date."),
	}, ""))

	endnotes := wordPartXML("endnotes", strings.Join([]string{
		`<w:endnote w:type="separator" w:id="-1"><w:p><w:r><w:separator/></w:r></w:p></w:endnote>`,
		`<w:endnote w:type="continuationSeparator" w:id="0"><w:p><w:r><w:continuationSeparator/></w:r></w:p></w:endnote>`,
		noteDefinition("endnote", "2", "This agreement is governed by the law of the supplier's seat."),
	}, ""))

	a := newArchive()
	a.addDocument(body)
	a.addXMLPart("word/footnotes.xml", contentTypeFootnotes, footnotes)
	a.addXMLPart("word/endnotes.xml", contentTypeEndnotes, endnotes)
	a.addRelationships("word/document.xml", []relationship{
		{id: "rId1", relType: relTypeFootnotes, target: "footnotes.xml"},
		{id: "rId2", relType: relTypeEndnotes, target: "endnotes.xml"},
	})
	return a.bytes()
}

// noteDefinition renders one footnote or endnote definition body.
func noteDefinition(kind, id, value string) string {
	return `<w:` + kind + ` w:id="` + id + `">` +
		styledParagraph("FootnoteText", text(value)) +
		`</w:` + kind + `>`
}

func buildStructuredTags() ([]byte, error) {
	body := strings.Join([]string{
		heading(1, "Content Controls"),
		structuredTag("Supplier name", "supplierName", paragraph(text("Northwind Services Ltd."))),
		structuredTag("Declaration", "declaration", strings.Join([]string{
			heading(2, "Supplier Declaration"),
			paragraph(text("The supplier confirms that all statements in this form are complete and correct.")),
		}, "")),
		paragraph(text("A content control may also wrap a table:")),
		structuredTag("Contact table", "contactTable", table(2, [][]cell{
			{boldCell("Field"), boldCell("Value")},
			{textCell("Registered office"), textCell("14 Harbour Road, Bristol")},
			{textCell("Company number"), textCell("08421337")},
		})),
		paragraph(text("A content control inside a table cell:")),
		table(2, [][]cell{
			{boldCell("Field"), boldCell("Value")},
			{
				textCell("Authorised signatory"),
				{blocks: structuredTag("Signatory", "signatory", paragraph(text("Jordan Ellis, Managing Director")))},
			},
		}),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	return a.bytes()
}

func buildFieldsAlternateContent() ([]byte, error) {
	choiceParagraph := paragraph(text("Modern rendering: the choice branch is preferred."))
	fallbackParagraph := paragraph(text("Legacy rendering: the fallback branch is ignored when a choice exists."))

	body := strings.Join([]string{
		heading(1, "Fields and Alternate Content"),
		paragraph(
			text("Document reference "),
			complexField(` REF _Ref12345 \h `, "NS-2026-0147"),
			text(" is generated by a complex field."),
		),
		paragraph(
			text("A hyperlink field renders its display value: "),
			complexField(` HYPERLINK "https://example.com/tender/NS-2026-0147" `, "tender detail"),
			text("."),
		),
		paragraph(
			text("Signature date: "),
			simpleField(` DATE \@ "d MMMM yyyy" `, "7 September 2026"),
			text("."),
		),
		paragraph(
			text("Nested fields keep only the outer display value: "),
			`<w:r><w:fldChar w:fldCharType="begin"/></w:r>`,
			`<w:r><w:instrText xml:space="preserve"> IF </w:instrText></w:r>`,
			`<w:r><w:fldChar w:fldCharType="begin"/></w:r>`,
			`<w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r>`,
			`<w:r><w:fldChar w:fldCharType="end"/></w:r>`,
			`<w:r><w:fldChar w:fldCharType="separate"/></w:r>`,
			text("first page"),
			`<w:r><w:fldChar w:fldCharType="end"/></w:r>`,
			text("."),
		),
		alternateContent("wps", choiceParagraph, fallbackParagraph),
		paragraph(
			text("Run-level alternate content: "),
			alternateContent("wps", text("choice run"), text("fallback run")),
			text("."),
		),
		sectionProperties(),
	}, "")

	a := newArchive()
	a.addDocument(body)
	return a.bytes()
}

func buildServiceAgreement() ([]byte, error) {
	pngData, err := pngBytes()
	if err != nil {
		return nil, err
	}

	body := strings.Join([]string{
		heading(1, "Service Agreement for the Reporting Platform"),
		paragraph(text("Contract reference NS-2026-0147, concluded under the framework agreement of 4 March 2025.")),
		heading(2, "1. Contracting Parties"),
		table(2, [][]cell{
			{boldCell("Client"), boldCell("Supplier")},
			{
				{blocks: paragraph(text("Harbour City Council")) +
					paragraph(text("14 Harbour Road, Bristol")) +
					paragraph(text("Company number 08421337"))},
				{blocks: paragraph(text("Northwind Services Ltd.")) +
					paragraph(text("2 Kingsway, Manchester")) +
					paragraph(text("Company number 11902884"))},
			},
			{
				{blocks: paragraph(text("Represented by Jordan Ellis, Head of Procurement"))},
				{blocks: paragraph(text("Represented by Priya Raman, Managing Director"))},
			},
		}),
		heading(2, "2. Subject of the Agreement"),
		paragraph(text("The supplier delivers implementation, hosting, and support services for the reporting platform described in Annex 1. The client pays the agreed price and provides the access and data needed for delivery.")),
		paragraph(
			text("The technical specification is published at "),
			hyperlink("rId6", styledText("the tender detail page", false, false, true)),
			text(" and forms an integral part of this agreement"),
			noteRun("footnote", "2"),
			text("."),
		),
		heading(2, "3. Scope of Services"),
		listParagraph(1, 0, text("Implementation services")),
		listParagraph(1, 1, text("Environment provisioning and configuration.")),
		listParagraph(1, 1, text("Migration of historical reporting data.")),
		listParagraph(1, 2, text("Validation of migrated records against the source system.")),
		listParagraph(1, 0, text("Operating services")),
		listParagraph(1, 1, text("Hosting with an availability target of 99.5 per cent per month.")),
		listParagraph(1, 1, text("Incident response within the agreed reaction times.")),
		listParagraph(1, 0, text("Support and training")),
		listParagraph(1, 1, text("Two training days for up to twelve users.")),
		heading(2, "4. Price and Payment Terms"),
		table(4, [][]cell{
			{{blocks: paragraph(styledText("Price summary in EUR excluding VAT", true, false, false)), gridSpan: 4}},
			{boldCell("Item"), boldCell("Quantity"), boldCell("Unit price"), boldCell("Total")},
			{
				{blocks: paragraph(text("Implementation")), vMerge: "restart"},
				textCell("1"),
				textCell("4,500.00"),
				textCell("4,500.00"),
			},
			{
				{vMerge: "continue"},
				textCell("2"),
				textCell("1,200.00"),
				textCell("2,400.00"),
			},
			{textCell("Annual hosting"), textCell("12"), textCell("240.00"), textCell("2,880.00")},
			{textCell("Support hours"), textCell("40"), textCell("75.00"), textCell("3,000.00")},
			{
				{blocks: paragraph(styledText("Total", true, false, false)), gridSpan: 3},
				{blocks: paragraph(styledText("12,780.00", true, false, false))},
			},
		}),
		paragraph(text("Invoices are payable within thirty days of delivery. Late payment interest follows the statutory rate.")),
		paragraph(
			text("The price is fixed for the first contract year"),
			noteRun("footnote", "3"),
			text("."),
		),
		heading(2, "5. Service Levels"),
		table(3, [][]cell{
			{boldCell("Severity"), boldCell("Reaction time"), boldCell("Resolution target")},
			{textCell("Critical"), textCell("1 hour"), textCell("8 business hours")},
			{textCell("High"), textCell("4 business hours"), textCell("2 business days")},
			{textCell("Standard"), textCell("1 business day"), textCell("10 business days")},
		}),
		heading(2, "6. Acceptance"),
		paragraph(text("Each milestone is accepted by a written protocol signed by both parties. The client may reject a delivery only for defects that prevent the agreed use.")),
		heading(3, "6.1 Acceptance Checklist"),
		listParagraph(2, 0, text("Test environment restored from the anonymised data set.")),
		listParagraph(2, 0, text("Acceptance test suite executed without critical defects.")),
		listParagraph(2, 0, text("Operating documentation handed over in English.")),
		heading(2, "7. Confidentiality and Personal Data"),
		paragraph(text("Each party protects confidential information of the other party and processes personal data only for the purposes of this agreement. The supplier notifies the client of any personal data breach without undue delay.")),
		heading(2, "8. Term and Termination"),
		paragraph(text("The agreement is concluded for twenty-four months from the acceptance of the first milestone. Either party may terminate for material breach that is not remedied within thirty days of written notice.")),
		heading(2, "9. Final Provisions"),
		paragraph(
			text("This agreement is signed electronically on "),
			simpleField(` DATE \@ "d MMMM yyyy" `, "7 September 2026"),
			text(" in two counterparts"),
			noteRun("footnote", "4"),
			text("."),
		),
		heading(2, "Annex 1 — Architecture Overview"),
		paragraph(text("The diagram below shows the platform components covered by this agreement.")),
		imageParagraph(20, "rId7", "architecture.png", "Architecture overview of the reporting platform"),
		structuredTag("Signature block", "signatureBlock", strings.Join([]string{
			paragraph(text("For the client:"), tabRun(), text("For the supplier:")),
			paragraph(text("Jordan Ellis"), tabRun(), text("Priya Raman")),
		}, "")),
		sectionProperties(
			headerReference("first", "rId3"),
			headerReference("default", "rId2"),
			footerReference("default", "rId4"),
		),
	}, "")

	defaultHeader := wordPartXML("hdr", paragraph(
		text("Service Agreement NS-2026-0147"),
		tabRun(),
		text("Harbour City Council"),
	))

	firstHeader := wordPartXML("hdr", strings.Join([]string{
		imageParagraph(30, "rId1", "logo.png", "Northwind Services logo"),
		paragraph(text("Northwind Services Ltd. — Reporting Platform")),
	}, ""))

	footer := wordPartXML("ftr", paragraph(
		text("Page "),
		complexField(" PAGE ", "1"),
		text(" of "),
		complexField(" NUMPAGES ", "9"),
		tabRun(),
		text("Restricted distribution"),
	))

	footnotes := wordPartXML("footnotes", strings.Join([]string{
		`<w:footnote w:type="separator" w:id="-1"><w:p><w:r><w:separator/></w:r></w:p></w:footnote>`,
		`<w:footnote w:type="continuationSeparator" w:id="0"><w:p><w:r><w:continuationSeparator/></w:r></w:p></w:footnote>`,
		noteDefinition("footnote", "2", "Annex 1 prevails over the body of the agreement in technical matters."),
		noteDefinition("footnote", "3", "Indexation may be agreed for the second contract year by written amendment."),
		noteDefinition("footnote", "4", "Each party receives one signed counterpart."),
	}, ""))

	a := newArchive()
	a.addDocument(body)
	a.addXMLPart("word/numbering.xml", contentTypeNumbering, numberingXML())
	a.addXMLPart("word/header1.xml", contentTypeHeader, defaultHeader)
	a.addXMLPart("word/header2.xml", contentTypeHeader, firstHeader)
	a.addXMLPart("word/footer1.xml", contentTypeFooter, footer)
	a.addXMLPart("word/footnotes.xml", contentTypeFootnotes, footnotes)
	a.addPart("word/media/logo.png", pngData)
	a.addPart("word/media/architecture.png", pngData)
	a.addRelationships("word/document.xml", []relationship{
		{id: "rId1", relType: relTypeNumbering, target: "numbering.xml"},
		{id: "rId2", relType: relTypeHeader, target: "header1.xml"},
		{id: "rId3", relType: relTypeHeader, target: "header2.xml"},
		{id: "rId4", relType: relTypeFooter, target: "footer1.xml"},
		{id: "rId5", relType: relTypeFootnotes, target: "footnotes.xml"},
		{id: "rId6", relType: relTypeHyperlink, target: "https://example.com/tender/NS-2026-0147", targetMode: "External"},
		{id: "rId7", relType: relTypeImage, target: "media/architecture.png"},
	})
	a.addRelationships("word/header2.xml", []relationship{
		{id: "rId1", relType: relTypeImage, target: "media/logo.png"},
	})
	return a.bytes()
}

// buildTruncated returns a valid package cut short so the zip central
// directory is missing.
func buildTruncated() ([]byte, error) {
	data, err := buildSimple()
	if err != nil {
		return nil, err
	}
	cut := len(data) * 2 / 3
	if cut == 0 {
		return nil, fmt.Errorf("source archive too small to truncate: %d bytes", len(data))
	}
	return data[:cut], nil
}
