package actions

import (
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/jung-kurt/gofpdf"
)

const (
	pdfLineHeight  = 6.0
	pdfBlockMargin = 4.0
)

// renderMarkdownToPDF walks the markdown AST and renders it to the PDF using tr for all text.
func renderMarkdownToPDF(pdf *gofpdf.Fpdf, tr func(string) string, doc ast.Node) {
	for child := ast.GetFirstChild(doc); child != nil; child = ast.GetNextNode(child) {
		renderBlock(pdf, tr, child)
	}
}

func renderBlock(pdf *gofpdf.Fpdf, tr func(string) string, node ast.Node) {
	switch n := node.(type) {
	case *ast.Document:
		for child := ast.GetFirstChild(n); child != nil; child = ast.GetNextNode(child) {
			renderBlock(pdf, tr, child)
		}
	case *ast.Heading:
		level := n.Level
		if level > 6 {
			level = 6
		}
		size := float64(22 - level*2)
		if size < 12 {
			size = 12
		}
		pdf.SetFont("Arial", "B", size)
		writeInlineContent(pdf, tr, n)
		pdf.Ln(pdfLineHeight + pdfBlockMargin)
		pdf.SetFont("Arial", "", 12)
	case *ast.Paragraph:
		writeInlineContent(pdf, tr, n)
		pdf.Ln(pdfLineHeight + pdfBlockMargin)
	case *ast.List:
		listType := n.ListFlags
		ordered := (listType & ast.ListTypeOrdered) != 0
		start := n.Start
		if start <= 0 {
			start = 1
		}
		itemNum := 0
		for child := ast.GetFirstChild(n); child != nil; child = ast.GetNextNode(child) {
			if item, ok := child.(*ast.ListItem); ok {
				itemNum++
				var bullet string
				if ordered {
					bullet = tr(fmt.Sprintf("%d. ", start+itemNum-1))
				} else {
					bullet = tr("• ")
				}
				pdf.SetFont("Arial", "", 12)
				pdf.CellFormat(8, pdfLineHeight, bullet, "", 0, "", false, 0, "")
				for inner := ast.GetFirstChild(item); inner != nil; inner = ast.GetNextNode(inner) {
					renderBlock(pdf, tr, inner)
				}
			}
		}
		pdf.Ln(pdfBlockMargin)
	case *ast.ListItem:
		for child := ast.GetFirstChild(n); child != nil; child = ast.GetNextNode(child) {
			renderBlock(pdf, tr, child)
		}
	case *ast.CodeBlock:
		pdf.SetFont("Courier", "", 10)
		lit := n.Literal
		if lit == nil {
			lit = n.Content
		}
		if len(lit) > 0 {
			pdf.MultiCell(0, pdfLineHeight-1, tr(string(lit)), "", "", false)
		}
		pdf.SetFont("Arial", "", 12)
		pdf.Ln(pdfBlockMargin)
	case *ast.BlockQuote:
		left, _, _, _ := pdf.GetMargins()
		saveLeft := left
		pdf.SetLeftMargin(saveLeft + 4)
		pdf.SetX(saveLeft + 4)
		for child := ast.GetFirstChild(n); child != nil; child = ast.GetNextNode(child) {
			renderBlock(pdf, tr, child)
		}
		pdf.SetLeftMargin(saveLeft)
		pdf.Ln(pdfBlockMargin)
	case *ast.HorizontalRule:
		pdf.Ln(pdfBlockMargin)
		pdf.Line(pdf.GetX(), pdf.GetY(), pdf.GetX()+190, pdf.GetY())
		pdf.Ln(pdfBlockMargin)
	case *ast.Table:
		renderTable(pdf, tr, n)
		pdf.Ln(pdfBlockMargin)
	case *ast.MathBlock:
		pdf.SetFont("Courier", "", 10)
		lit := n.Literal
		if lit == nil {
			lit = n.Content
		}
		if len(lit) > 0 {
			pdf.MultiCell(0, pdfLineHeight-1, tr(string(lit)), "", "", false)
		}
		pdf.SetFont("Arial", "", 12)
		pdf.Ln(pdfBlockMargin)
	case *ast.HTMLBlock:
		lit := n.Literal
		if lit == nil {
			lit = n.Content
		}
		if len(lit) > 0 {
			pdf.SetFont("Courier", "", 9)
			pdf.MultiCell(0, pdfLineHeight-1, tr(string(lit)), "", "", false)
			pdf.SetFont("Arial", "", 12)
		}
		pdf.Ln(pdfBlockMargin)
	case *ast.Aside:
		left, _, _, _ := pdf.GetMargins()
		saveLeft := left
		pdf.SetLeftMargin(saveLeft + 4)
		pdf.SetX(saveLeft + 4)
		for child := ast.GetFirstChild(n); child != nil; child = ast.GetNextNode(child) {
			renderBlock(pdf, tr, child)
		}
		pdf.SetLeftMargin(saveLeft)
		pdf.Ln(pdfBlockMargin)
	default:
		// Unknown block: try to render as inline content (e.g. paragraph-like)
		if ast.GetFirstChild(node) != nil {
			writeInlineContent(pdf, tr, node)
			pdf.Ln(pdfLineHeight + pdfBlockMargin)
		}
	}
}

const (
	pdfTableLineHt   = 7.0
	pdfTableHeaderR  = 72
	pdfTableHeaderG  = 72
	pdfTableHeaderB  = 72
	pdfTableBorderR  = 200
	pdfTableBorderG  = 200
	pdfTableBorderB  = 200
	pdfTableStripR   = 248
	pdfTableStripG   = 248
	pdfTableStripB   = 248
)

// renderTable draws a markdown table. Table contains TableHeader and TableBody, each with TableRows of TableCells.
func renderTable(pdf *gofpdf.Fpdf, tr func(string) string, table *ast.Table) {
	left, _, right, _ := pdf.GetMargins()
	pageW := 210.0
	tblW := pageW - left - right

	// Collect all rows: header rows first, then body (and footer if any)
	var rows [][]string
	var numCols int
	for section := ast.GetFirstChild(table); section != nil; section = ast.GetNextNode(section) {
		for rowNode := ast.GetFirstChild(section); rowNode != nil; rowNode = ast.GetNextNode(rowNode) {
			row, ok := rowNode.(*ast.TableRow)
			if !ok {
				continue
			}
			var cells []string
			for c := ast.GetFirstChild(row); c != nil; c = ast.GetNextNode(c) {
				if cell, ok := c.(*ast.TableCell); ok {
					cells = append(cells, tr(getCellText(cell)))
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
				if len(cells) > numCols {
					numCols = len(cells)
				}
			}
		}
	}
	if numCols == 0 {
		return
	}
	colW := tblW / float64(numCols)
	lineHt := pdfTableLineHt

	// Save current colors and set light gray borders for the table
	saveDrawR, saveDrawG, saveDrawB := pdf.GetDrawColor()
	saveFillR, saveFillG, saveFillB := pdf.GetFillColor()
	saveTextR, saveTextG, saveTextB := pdf.GetTextColor()
	pdf.SetDrawColor(pdfTableBorderR, pdfTableBorderG, pdfTableBorderB)

	for i, row := range rows {
		isHeader := i == 0
		lastRow := i == len(rows) - 1
		// Header: dark gray background, white text, bold
		if isHeader {
			pdf.SetFont("Arial", "B", 12)
			pdf.SetFillColor(pdfTableHeaderR, pdfTableHeaderG, pdfTableHeaderB)
			pdf.SetTextColor(255, 255, 255)
		} else {
			pdf.SetFont("Arial", "", 12)
			pdf.SetTextColor(0, 0, 0)
			if i%2 == 1 {
				pdf.SetFillColor(pdfTableStripR, pdfTableStripG, pdfTableStripB)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
		}
		border := "LTR"
		if lastRow {
			border = "LTRB"
		}
		fill := true
		for j, cellText := range row {
			w := colW
			if j == numCols-1 {
				w = 0
			}
			pdf.CellFormat(w, lineHt, cellText, border, 0, "L", fill, 0, "")
		}
		pdf.Ln(lineHt)
	}

	// Restore colors and font
	pdf.SetDrawColor(saveDrawR, saveDrawG, saveDrawB)
	pdf.SetFillColor(saveFillR, saveFillG, saveFillB)
	pdf.SetTextColor(saveTextR, saveTextG, saveTextB)
	pdf.SetFont("Arial", "", 12)
}

// getInlineText returns plain text from an inline container (e.g. Image alt text).
func getInlineText(node ast.Node) string {
	var b []byte
	for child := ast.GetFirstChild(node); child != nil; child = ast.GetNextNode(child) {
		if leaf, ok := child.(*ast.Leaf); ok && len(leaf.Literal) > 0 {
			b = append(b, leaf.Literal...)
		} else if text, ok := child.(*ast.Text); ok {
			lit := text.Literal
			if lit == nil {
				lit = text.Content
			}
			if len(lit) > 0 {
				b = append(b, lit...)
			}
		} else {
			b = append(b, getInlineText(child)...)
		}
	}
	return string(b)
}

// getCellText returns plain text from a table cell (walks Paragraph/Text and Leaf nodes).
func getCellText(node ast.Node) string {
	var b []byte
	for child := ast.GetFirstChild(node); child != nil; child = ast.GetNextNode(child) {
		if leaf, ok := child.(*ast.Leaf); ok && len(leaf.Literal) > 0 {
			b = append(b, leaf.Literal...)
		} else if text, ok := child.(*ast.Text); ok {
			lit := text.Literal
			if lit == nil {
				lit = text.Content
			}
			if len(lit) > 0 {
				b = append(b, lit...)
			}
		} else {
			b = append(b, getCellText(child)...)
		}
	}
	return string(b)
}

// writeInlineContent outputs inline content (text, strong, emph, code) with correct font changes.
func writeInlineContent(pdf *gofpdf.Fpdf, tr func(string) string, node ast.Node) {
	lineHt := pdfLineHeight
	left, _, right, _ := pdf.GetMargins()
	pageW := 210.0 // A4 mm
	maxW := pageW - left - right

	for child := ast.GetFirstChild(node); child != nil; child = ast.GetNextNode(child) {
		writeInline(pdf, tr, child, lineHt, maxW)
	}
}

func writeInline(pdf *gofpdf.Fpdf, tr func(string) string, node ast.Node, lineHt, maxW float64) {
	switch n := node.(type) {
	case *ast.Text:
		lit := n.Literal
		if lit == nil {
			lit = n.Content
		}
		if len(lit) > 0 {
			cellWrap(pdf, tr(string(lit)), lineHt, maxW)
		}
	case *ast.Strong:
		pdf.SetFont("Arial", "B", 12)
		for c := ast.GetFirstChild(n); c != nil; c = ast.GetNextNode(c) {
			writeInline(pdf, tr, c, lineHt, maxW)
		}
		pdf.SetFont("Arial", "", 12)
	case *ast.Emph:
		pdf.SetFont("Arial", "I", 12)
		for c := ast.GetFirstChild(n); c != nil; c = ast.GetNextNode(c) {
			writeInline(pdf, tr, c, lineHt, maxW)
		}
		pdf.SetFont("Arial", "", 12)
	case *ast.Code:
		lit := n.Literal
		if lit == nil {
			lit = n.Content
		}
		if len(lit) > 0 {
			pdf.SetFont("Courier", "", 11)
			cellWrap(pdf, tr(string(lit)), lineHt, maxW)
			pdf.SetFont("Arial", "", 12)
		}
	case *ast.Link:
		for c := ast.GetFirstChild(n); c != nil; c = ast.GetNextNode(c) {
			writeInline(pdf, tr, c, lineHt, maxW)
		}
		if len(n.Destination) > 0 {
			pdf.SetFont("Arial", "I", 10)
			cellWrap(pdf, tr(" ("+string(n.Destination)+")"), lineHt, maxW)
			pdf.SetFont("Arial", "", 12)
		}
	case *ast.Image:
		alt := getInlineText(n)
		if alt != "" {
			cellWrap(pdf, tr(alt), lineHt, maxW)
		}
		if len(n.Destination) > 0 {
			pdf.SetFont("Arial", "I", 10)
			cellWrap(pdf, tr(" [Image: "+string(n.Destination)+"]"), lineHt, maxW)
			pdf.SetFont("Arial", "", 12)
		}
	case *ast.Del:
		for c := ast.GetFirstChild(n); c != nil; c = ast.GetNextNode(c) {
			writeInline(pdf, tr, c, lineHt, maxW)
		}
	case *ast.Subscript:
		pdf.SetFont("Arial", "", 9)
		for c := ast.GetFirstChild(n); c != nil; c = ast.GetNextNode(c) {
			writeInline(pdf, tr, c, lineHt, maxW)
		}
		pdf.SetFont("Arial", "", 12)
	case *ast.Superscript:
		pdf.SetFont("Arial", "", 9)
		for c := ast.GetFirstChild(n); c != nil; c = ast.GetNextNode(c) {
			writeInline(pdf, tr, c, lineHt, maxW)
		}
		pdf.SetFont("Arial", "", 12)
	case *ast.Math:
		lit := n.Literal
		if lit == nil {
			lit = n.Content
		}
		if len(lit) > 0 {
			pdf.SetFont("Courier", "", 10)
			cellWrap(pdf, tr(string(lit)), lineHt, maxW)
			pdf.SetFont("Arial", "", 12)
		}
	case *ast.Hardbreak:
		pdf.Ln(lineHt)
	case *ast.Softbreak:
		pdf.Ln(lineHt)
	default:
		if leaf, ok := node.(*ast.Leaf); ok && len(leaf.Literal) > 0 {
			cellWrap(pdf, tr(string(leaf.Literal)), lineHt, maxW)
		}
	}
}

// cellWrap outputs text with word-wrap: splits on spaces and starts a new line when the next word would overflow.
func cellWrap(pdf *gofpdf.Fpdf, s string, lineHt, maxW float64) {
	left, _, _, _ := pdf.GetMargins()
	words := strings.Fields(s)
	for i, word := range words {
		wordW := pdf.GetStringWidth(word)
		spaceW := 0.0
		if i > 0 {
			spaceW = pdf.GetStringWidth(" ")
		}
		x := pdf.GetX()
		// If this word (and preceding space) would overflow, start a new line first.
		if i > 0 {
			if x+spaceW+wordW > maxW && x > left {
				pdf.Ln(lineHt)
				x = pdf.GetX()
			} else {
				pdf.CellFormat(spaceW, lineHt, " ", "", 0, "", false, 0, "")
				x = pdf.GetX()
			}
		} else if wordW > 0 && x+wordW > maxW && x > left {
			pdf.Ln(lineHt)
			x = pdf.GetX()
		}
		// Single word longer than line width: use MultiCell so it wraps.
		if wordW > maxW-left {
			pdf.MultiCell(0, lineHt, word, "", "", false)
		} else {
			if x+wordW > maxW && x > left {
				pdf.Ln(lineHt)
			}
			pdf.CellFormat(wordW, lineHt, word, "", 0, "", false, 0, "")
		}
	}
}
