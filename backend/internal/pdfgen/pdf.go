// Package pdfgen renders Devis Generator : VIBED documents to PDF.
//
// La mise en page reprend l’aperçu A4 (couleurs, hiérarchie, colonnes).
// Les polices Go sont embarquées en UTF-8 : Helvetica (WinAnsi) cassait
// les accents, l’euro et les apostrophes typographiques, et ne pouvait
// ni poser le logo ni suivre la maquette.
package pdfgen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/phpdave11/gofpdf"
	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
)

type rgb struct{ R, G, B int }

var (
	colorInk    = rgb{27, 58, 75}
	colorMuted  = rgb{91, 107, 120}
	colorAccent = rgb{47, 158, 148}
	colorBorder = rgb{217, 225, 231}
	colorLine   = rgb{227, 233, 237}
	colorHead   = rgb{238, 243, 245}
	colorDash   = rgb{195, 208, 219}
)

type styled struct {
	text  string
	style string
	size  float64
	lh    float64
	col   rgb
}

type col struct {
	title string
	w     float64
	align string
}

type richLine struct {
	text  string
	muted bool
}

type sheet struct {
	pdf      *gofpdf.Fpdf
	pageW    float64
	pageH    float64
	marginL  float64
	marginR  float64
	marginT  float64
	marginB  float64
	contentW float64
	running  string
	foot     string
}

// Render builds an A4 PDF for a document and its parties.
func Render(doc biz.Document, issuer biz.IssuerProfile, client biz.Client, mentions legal.Mentions) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	title := displayTitle(doc.Kind)
	pdf.SetTitle(title, true)
	pdf.SetAuthor(firstNonEmpty(issuer.LegalName, "Devis Generator : VIBED"), true)
	pdf.SetCreator("Devis Generator : VIBED", true)
	pdf.SetCompression(true)

	if err := loadFonts(pdf); err != nil {
		return nil, err
	}
	mark, err := logoPNG()
	if err != nil {
		return nil, err
	}
	if pdf.RegisterImageOptionsReader("mark", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(mark)) == nil {
		if err := pdf.Error(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("logo non enregistré")
	}

	pageW, pageH := pdf.GetPageSize()
	s := &sheet{
		pdf: pdf, pageW: pageW, pageH: pageH,
		marginL: 16, marginR: 16, marginT: 15, marginB: 18,
		running: runningHeader(doc),
		foot:    footerLabel(issuer),
	}
	s.contentW = pageW - s.marginL - s.marginR

	pdf.SetMargins(s.marginL, s.marginT, s.marginR)
	pdf.SetCellMargin(0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AliasNbPages("")
	pdf.SetFooterFunc(s.footer)
	pdf.AddPage()

	s.header(doc, issuer)
	s.gap(3.2)
	s.accent(15)
	s.gap(5)
	if s.room() < 36 {
		s.newPage()
	}
	s.parties(doc, client)
	s.gap(6)
	s.table(doc)
	s.gap(5)
	s.totals(doc)
	if strings.TrimSpace(doc.Notes) != "" {
		s.gap(4)
		s.prose(doc.Notes, 9, 4.3, colorInk)
	}
	if items := mentionLines(doc, issuer, mentions); len(items) > 0 {
		s.gap(3.5)
		s.paragraphs(items, 8, 3.7, colorInk)
	}
	if doc.Kind == "quote" {
		s.signature()
	}

	if err := pdf.Error(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func loadFonts(pdf *gofpdf.Fpdf) error {
	pdf.AddUTF8FontFromBytes("DG", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("DG", "B", gobold.TTF)
	pdf.AddUTF8FontFromBytes("DGM", "", gomedium.TTF)
	pdf.SetFont("DG", "", 10)
	if err := pdf.Error(); err != nil {
		return fmt.Errorf("polices: %w", err)
	}
	if pdf.GetStringWidth("Été €") < 2 {
		return fmt.Errorf("polices UTF-8 inutilisables")
	}
	return nil
}

func (s *sheet) header(doc biz.Document, issuer biz.IssuerProfile) {
	y := s.pdf.GetY()
	const logo = 10.0
	s.pdf.ImageOptions("mark", s.marginL, y, logo, logo, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	rightW := 78.0
	if rightW > s.contentW*0.46 {
		rightW = s.contentW * 0.46
	}
	leftX := s.marginL + logo + 3
	leftW := s.contentW - rightW - (logo + 6)
	if leftW < 40 {
		leftW = 40
	}
	franchise := doc.VATRegime == billing.RegimeFranchise293B
	yLeft := s.writeBlock(leftX, y, leftW, issuerRows(issuer, franchise), "L")

	rightX := s.marginL + s.contentW - rightW
	yTitle := s.drawTitle(rightX+rightW, y-0.4, rightW, displayTitle(doc.Kind))
	num := "Numéro attribué à l\u2019émission"
	if doc.Number != nil && strings.TrimSpace(*doc.Number) != "" {
		num = "N° " + strings.TrimSpace(*doc.Number)
	}
	rows := []styled{{text: num, style: "", size: 9, lh: 4.3, col: colorMuted}}
	for _, m := range metaLines(doc) {
		rows = append(rows, styled{text: m, style: "", size: 9, lh: 4.3, col: colorInk})
	}
	yRight := s.writeBlock(rightX, yTitle, rightW, rows, "R")

	end := y + logo
	if yLeft > end {
		end = yLeft
	}
	if yRight > end {
		end = yRight
	}
	s.pdf.SetY(end)
}

func (s *sheet) parties(doc biz.Document, client biz.Client) {
	y := s.pdf.GetY()
	boxW := 72.0
	gap := 8.0
	leftW := s.contentW - boxW - gap
	title := strings.TrimSpace(doc.Title)
	if title == "" {
		title = "\u2014"
	}
	yLeft := s.writeBlock(s.marginL, y, leftW, []styled{
		{text: "OBJET", style: "M", size: 7.5, lh: 3.8, col: colorMuted},
		{text: title, style: "M", size: 10.5, lh: 5, col: colorInk},
	}, "L")

	heading := "FACTURÉ À"
	if doc.Kind == "quote" {
		heading = "ADRESSÉ À"
	}
	name := strings.TrimSpace(client.Name)
	if name == "" {
		name = "[Nom du client]"
	}
	rows := []styled{
		{text: heading, style: "M", size: 7.5, lh: 3.6, col: colorMuted},
		{text: name, style: "M", size: 9.5, lh: 4.6, col: colorInk},
	}
	for _, ln := range addressLines(client.Address) {
		rows = append(rows, styled{text: ln, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if v := strings.TrimSpace(client.SIREN); v != "" {
		rows = append(rows, styled{text: "SIREN " + v, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if v := strings.TrimSpace(client.SIRET); v != "" {
		rows = append(rows, styled{text: "SIRET " + v, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if doc.VATRegime != billing.RegimeFranchise293B {
		if v := strings.TrimSpace(client.VATNumber); v != "" {
			rows = append(rows, styled{text: "TVA " + v, style: "", size: 9, lh: 4.1, col: colorMuted})
		}
	}
	if v := strings.TrimSpace(client.Email); v != "" {
		rows = append(rows, styled{text: v, style: "", size: 9, lh: 4.1, col: colorMuted})
	}

	pad := 3.2
	innerW := boxW - 2*pad
	h := s.measure(rows, innerW) + 2*pad
	if h < 24 {
		h = 24
	}
	x := s.marginL + s.contentW - boxW
	s.setDraw(colorBorder)
	s.pdf.SetLineWidth(0.3)
	s.pdf.RoundedRect(x, y, boxW, h, 1.1, "1234", "D")
	s.writeBlock(x+pad, y+pad, innerW, rows, "L")

	end := yLeft
	if y+h > end {
		end = y + h
	}
	s.pdf.SetY(end)
}

func (s *sheet) table(doc biz.Document) {
	franchise := doc.VATRegime == billing.RegimeFranchise293B
	cols := tableCols(franchise, s.contentW)
	if s.room() < 22 {
		s.newPage()
	}
	s.drawHead(cols)
	if len(doc.Lines) == 0 {
		s.drawEmptyRow(cols)
		return
	}
	for _, line := range doc.Lines {
		s.drawLine(cols, line, franchise)
	}
}

func (s *sheet) drawEmptyRow(cols []col) {
	const h = 8.0
	y := s.pdf.GetY()
	s.setDraw(colorLine)
	s.pdf.SetLineWidth(0.15)
	s.pdf.Line(s.marginL, y+h, s.marginL+s.contentW, y+h)
	s.font("", 9)
	s.setText(colorMuted)
	s.pdf.SetXY(s.marginL+1.6, y+1.8)
	s.pdf.CellFormat(cols[0].w, 4.2, "\u2014", "", 0, "L", false, 0, "")
	s.pdf.SetY(y + h)
}

func (s *sheet) drawLine(cols []col, line biz.DocLine, franchise bool) {
	s.font("", 9)
	descW := cols[0].w - 3.2
	raw := strings.TrimSpace(clean(line.Description))
	if raw == "" {
		raw = "\u2014"
	}
	var rich []richLine
	for _, ln := range s.wrapped(raw, descW) {
		rich = append(rich, richLine{text: ln})
	}
	if line.DiscountBps > 0 {
		rich = append(rich, richLine{text: "Remise " + formatRate(line.DiscountBps), muted: true})
	}
	if len(rich) == 0 {
		rich = []richLine{{text: "\u2014"}}
	}

	values := []string{formatQty(line.QuantityMilli), formatEUR(line.UnitPriceCents)}
	if !franchise {
		values = append(values, formatRate(line.TaxRateBps))
	}
	values = append(values, formatEUR(line.TotalCents))

	const lh = 4.3
	const pad = 1.45
	// Une ligne qui tient sur une page vierge n’est pas coupée : sinon la fin
	// de la désignation se retrouve seule, sans montant, en haut de la page suivante.
	rowH := pad*2 + float64(len(rich))*lh
	if rowH <= 230 && s.room() < rowH {
		s.newPage()
		s.drawHead(cols)
	}
	i := 0
	first := true
	for guard := 0; i < len(rich) && guard < 5000; guard++ {
		nFit := int((s.room() - pad*2) / lh)
		if nFit < 1 {
			s.newPage()
			s.drawHead(cols)
			continue
		}
		n := len(rich) - i
		if n > nFit {
			n = nFit
		}
		h := pad*2 + float64(n)*lh
		y := s.pdf.GetY()
		s.setDraw(colorLine)
		s.pdf.SetLineWidth(0.15)
		s.pdf.Line(s.marginL, y+h, s.marginL+s.contentW, y+h)
		for k := 0; k < n; k++ {
			ln := rich[i+k]
			if ln.muted {
				s.setText(colorMuted)
			} else {
				s.setText(colorInk)
			}
			s.font("", 9)
			s.pdf.SetXY(s.marginL+1.6, y+pad+float64(k)*lh)
			s.pdf.CellFormat(descW, lh, ln.text, "", 0, "L", false, 0, "")
		}
		if first {
			x := s.marginL + cols[0].w
			for vi, val := range values {
				c := cols[vi+1]
				s.cell(x, y+pad, c.w-1.6, lh, val, "R", "", 9, colorInk)
				x += c.w
			}
			first = false
		}
		i += n
		s.pdf.SetY(y + h)
		if i < len(rich) {
			s.newPage()
			s.drawHead(cols)
		}
	}
}

func (s *sheet) drawHead(cols []col) {
	if s.room() < 10 {
		s.newPage()
	}
	y := s.pdf.GetY()
	const h = 7.2
	s.setFill(colorHead)
	s.pdf.Rect(s.marginL, y, s.contentW, h, "F")
	s.font("M", 7.5)
	s.setText(colorMuted)
	x := s.marginL
	for _, c := range cols {
		if c.align == "R" {
			s.pdf.SetXY(x, y+1.6)
			s.pdf.CellFormat(c.w-1.6, 4, c.title, "", 0, "R", false, 0, "")
		} else {
			s.pdf.SetXY(x+1.6, y+1.6)
			s.pdf.CellFormat(c.w-1.6, 4, c.title, "", 0, "L", false, 0, "")
		}
		x += c.w
	}
	s.setDraw(colorBorder)
	s.pdf.SetLineWidth(0.25)
	s.pdf.Line(s.marginL, y+h, s.marginL+s.contentW, y+h)
	s.pdf.SetY(y + h)
}

func (s *sheet) totals(doc biz.Document) {
	type kv struct {
		k, v string
		bold bool
	}
	rows := []kv{{k: "Total HT", v: formatEUR(doc.SubtotalCents)}}
	if doc.VATRegime == billing.RegimeFranchise293B {
		rows = append(rows, kv{k: "TVA", v: "\u2014"})
	} else if buckets := vatBuckets(doc); len(buckets) == 0 {
		rows = append(rows, kv{k: "TVA", v: formatEUR(doc.TaxCents)})
	} else {
		for _, b := range buckets {
			rows = append(rows, kv{k: "TVA " + formatRate(b.TaxRateBps), v: formatEUR(b.TaxCents)})
		}
	}
	rows = append(rows, kv{k: finalLabel(doc.Kind), v: formatEUR(doc.TotalCents), bold: true})

	need := float64(len(rows))*5.4 + 4
	if s.room() < need {
		s.newPage()
	}
	const blockW = 78.0
	x := s.marginL + s.contentW - blockW
	for _, row := range rows {
		if row.bold {
			y := s.pdf.GetY() + 1.1
			s.setDraw(colorInk)
			s.pdf.SetLineWidth(0.4)
			s.pdf.Line(x, y, x+blockW, y)
			s.pdf.SetY(y + 1.5)
			s.cell(x, s.pdf.GetY(), blockW*0.62, 6, row.k, "L", "B", 10.5, colorInk)
			s.cell(x+blockW*0.38, s.pdf.GetY(), blockW*0.62, 6, row.v, "R", "B", 10.5, colorInk)
			s.pdf.SetY(s.pdf.GetY() + 6.2)
			continue
		}
		y := s.pdf.GetY()
		s.cell(x, y, blockW*0.58, 5, row.k, "L", "", 9, colorMuted)
		s.cell(x+blockW*0.42, y, blockW*0.58, 5, row.v, "R", "", 9, colorInk)
		s.pdf.SetY(y + 5.1)
	}
}

func (s *sheet) signature() {
	const (
		sigW = 68.0
		sigH = 26.0
	)
	if s.pdf.GetY()+6+sigH > s.limitY() {
		s.newPage()
	}
	y := s.limitY() - sigH
	if y < s.pdf.GetY()+4 {
		y = s.pdf.GetY() + 4
	}
	x := s.marginL + s.contentW - sigW
	s.setDraw(colorDash)
	s.pdf.SetLineWidth(0.3)
	s.pdf.SetDashPattern([]float64{1.1, 1.1}, 0)
	s.pdf.RoundedRect(x, y, sigW, sigH, 1.1, "1234", "D")
	s.pdf.SetDashPattern([]float64{}, 0)
	s.pdf.SetLineWidth(0.2)
	s.font("", 8)
	s.setText(colorMuted)
	s.pdf.SetXY(x+3, y+2.4)
	s.pdf.CellFormat(sigW-6, 4, "Bon pour accord : date et signature", "", 0, "L", false, 0, "")
	s.pdf.SetY(y + sigH)
}

func (s *sheet) footer() {
	y := s.pageH - 13
	s.setDraw(colorLine)
	s.pdf.SetLineWidth(0.2)
	s.pdf.SetDashPattern([]float64{}, 0)
	s.pdf.Line(s.marginL, y, s.marginL+s.contentW, y)
	s.pdf.ImageOptions("mark", s.marginL, y+2.1, 4.2, 4.2, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	label := clean(s.foot)
	size := 7.5
	for size > 6 {
		s.font("", size)
		if s.pdf.GetStringWidth(label) <= s.contentW-28 {
			break
		}
		size -= 0.5
	}
	s.font("", size)
	s.setText(colorMuted)
	s.pdf.SetXY(s.marginL+6.2, y+2.15)
	s.pdf.CellFormat(s.contentW-24, 4.2, label, "", 0, "C", false, 0, "")
	s.font("", 7.5)
	s.pdf.SetXY(s.marginL+s.contentW-16, y+2.15)
	s.pdf.CellFormat(16, 4.2, fmt.Sprintf("%d / {nb}", s.pdf.PageNo()), "", 0, "R", false, 0, "")
}

func (s *sheet) prose(text string, size, lh float64, col rgb) {
	text = clean(text)
	if strings.TrimSpace(text) == "" {
		return
	}
	for _, para := range strings.Split(text, "\n") {
		if strings.TrimSpace(para) == "" {
			s.gap(lh * 0.55)
			continue
		}
		s.font("", size)
		for _, ln := range s.wrapped(para, s.contentW) {
			if s.room() < lh {
				s.newPage()
			}
			s.font("", size)
			s.setText(col)
			s.pdf.SetX(s.marginL)
			s.pdf.CellFormat(s.contentW, lh, ln, "", 1, "L", false, 0, "")
		}
	}
}

func (s *sheet) paragraphs(items []string, size, lh float64, col rgb) {
	for i, item := range items {
		s.prose(item, size, lh, col)
		if i < len(items)-1 {
			s.gap(0.7)
		}
	}
}

func (s *sheet) writeBlock(x, y, w float64, rows []styled, align string) float64 {
	yy := y
	for _, row := range rows {
		s.font(row.style, row.size)
		s.setText(row.col)
		lines := s.wrapped(row.text, w)
		if len(lines) == 0 {
			lines = []string{""}
		}
		for _, ln := range lines {
			s.font(row.style, row.size)
			s.setText(row.col)
			s.pdf.SetXY(x, yy)
			s.pdf.CellFormat(w, row.lh, ln, "", 0, align, false, 0, "")
			yy += row.lh
		}
	}
	return yy
}

func (s *sheet) measure(rows []styled, w float64) float64 {
	h := 0.0
	for _, row := range rows {
		s.font(row.style, row.size)
		n := len(s.wrapped(row.text, w))
		if n == 0 {
			n = 1
		}
		h += float64(n) * row.lh
	}
	return h
}

func (s *sheet) cell(x, y, w, h float64, text, align, style string, size float64, col rgb) {
	text = clean(text)
	sz := size
	for sz > 6 {
		s.font(style, sz)
		if s.pdf.GetStringWidth(text) <= w {
			break
		}
		sz -= 0.4
	}
	s.font(style, sz)
	s.setText(col)
	s.pdf.SetXY(x, y)
	s.pdf.CellFormat(w, h, text, "", 0, align, false, 0, "")
}

func (s *sheet) wrapped(text string, w float64) []string {
	text = clean(text)
	if w < 8 {
		w = 8
	}
	if text == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(text, "\n") {
		if strings.TrimSpace(part) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, s.pdf.SplitText(part, w)...)
	}
	return out
}

func (s *sheet) font(style string, size float64) {
	switch style {
	case "B":
		s.pdf.SetFont("DG", "B", size)
	case "M":
		s.pdf.SetFont("DGM", "", size)
	default:
		s.pdf.SetFont("DG", "", size)
	}
}

func (s *sheet) setFill(c rgb)   { s.pdf.SetFillColor(c.R, c.G, c.B) }
func (s *sheet) setDraw(c rgb)   { s.pdf.SetDrawColor(c.R, c.G, c.B) }
func (s *sheet) setText(c rgb)   { s.pdf.SetTextColor(c.R, c.G, c.B) }
func (s *sheet) limitY() float64 { return s.pageH - s.marginB }
func (s *sheet) room() float64   { return s.limitY() - s.pdf.GetY() }

func (s *sheet) gap(h float64) {
	if h <= 0 || s.pdf.GetY()+h > s.limitY() {
		return
	}
	s.pdf.SetY(s.pdf.GetY() + h)
}

func (s *sheet) accent(w float64) {
	y := s.pdf.GetY()
	s.setFill(colorAccent)
	s.pdf.Rect(s.marginL, y, w, 0.9, "F")
	s.pdf.SetY(y + 0.9)
}

func (s *sheet) newPage() {
	s.pdf.AddPage()
	s.pdf.SetXY(s.marginL, s.marginT)
	s.font("M", 8.5)
	s.setText(colorMuted)
	s.pdf.CellFormat(s.contentW, 4.6, clean(s.running)+"  \u2014  suite", "", 1, "L", false, 0, "")
	s.gap(1.2)
	s.accent(12)
	s.gap(3)
}

func tableCols(franchise bool, contentW float64) []col {
	qty, pu, tot := 16.0, 30.0, 32.0
	if franchise {
		return []col{
			{"DÉSIGNATION", contentW - qty - pu - tot, "L"},
			{"QTÉ", qty, "R"},
			{"PU HT", pu, "R"},
			{"TOTAL HT", tot, "R"},
		}
	}
	tva := 16.0
	return []col{
		{"DÉSIGNATION", contentW - qty - pu - tva - tot, "L"},
		{"QTÉ", qty, "R"},
		{"PU HT", pu, "R"},
		{"TVA", tva, "R"},
		{"TOTAL HT", tot, "R"},
	}
}

func (s *sheet) drawTitle(right, y, maxW float64, title string) float64 {
	size := 16.5
	track := 0.06
	for size > 11 && s.trackedWidth(title, size, track) > maxW {
		if track > 0 {
			track = 0
			continue
		}
		size -= 0.5
	}
	s.font("B", size)
	s.setText(colorInk)
	gap := size * 0.352778 * track
	rs := []rune(title)
	x := right - s.trackedWidth(title, size, track)
	const lh = 7.6
	for _, r := range rs {
		ch := string(r)
		cw := s.pdf.GetStringWidth(ch)
		s.pdf.SetXY(x, y)
		s.pdf.CellFormat(cw+0.3, lh, ch, "", 0, "L", false, 0, "")
		x += cw + gap
	}
	return y + lh
}

func (s *sheet) trackedWidth(str string, size, em float64) float64 {
	s.font("B", size)
	gap := size * 0.352778 * em
	w := 0.0
	rs := []rune(str)
	for _, r := range rs {
		w += s.pdf.GetStringWidth(string(r))
	}
	if len(rs) > 1 {
		w += gap * float64(len(rs)-1)
	}
	return w
}

func issuerRows(issuer biz.IssuerProfile, franchise bool) []styled {
	name := strings.TrimSpace(issuer.LegalName)
	if name == "" {
		name = "[Votre raison sociale]"
	}
	out := []styled{{text: name, style: "M", size: 11.5, lh: 5.1, col: colorInk}}
	if trade := strings.TrimSpace(issuer.TradeName); trade != "" && trade != name {
		out = append(out, styled{text: trade, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	for _, ln := range addressLines(issuer.Address) {
		out = append(out, styled{text: ln, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if p := strings.TrimSpace(issuer.Phone); p != "" {
		out = append(out, styled{text: "Tél. " + p, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if e := strings.TrimSpace(issuer.Email); e != "" {
		out = append(out, styled{text: e, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if v := strings.TrimSpace(issuer.SIRET); v != "" {
		out = append(out, styled{text: "SIRET " + v, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if !franchise {
		if v := strings.TrimSpace(issuer.VATNumber); v != "" {
			out = append(out, styled{text: "TVA " + v, style: "", size: 9, lh: 4.1, col: colorMuted})
		}
	}
	if v := strings.TrimSpace(issuer.RCS); v != "" {
		out = append(out, styled{text: "RCS " + v, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	if v := strings.TrimSpace(issuer.APECode); v != "" {
		out = append(out, styled{text: "APE " + v, style: "", size: 9, lh: 4.1, col: colorMuted})
	}
	return out
}

func metaLines(doc biz.Document) []string {
	date := "\u2014"
	if doc.IssueDate != nil {
		date = formatDate(*doc.IssueDate)
	}
	out := []string{"Date : " + date}
	if doc.Kind != "credit_note" && doc.DueDate != nil && strings.TrimSpace(*doc.DueDate) != "" {
		label := "Échéance"
		if doc.Kind == "quote" {
			label = "Valable jusqu\u2019au"
		}
		out = append(out, label+" : "+formatDate(*doc.DueDate))
	}
	return out
}

func mentionLines(doc biz.Document, issuer biz.IssuerProfile, mentions legal.Mentions) []string {
	var out []string
	isQuote := doc.Kind == "quote"
	if doc.VATRegime == billing.RegimeFranchise293B {
		if s := ensurePeriod(mentions.Franchise293B); s != "" {
			out = append(out, s)
		}
	}
	terms := strings.TrimSpace(doc.PaymentTerms)
	if terms == "" && !isQuote {
		terms = strings.TrimSpace(mentions.PaymentTermsDefault)
	}
	if terms != "" {
		line := strings.TrimRight(strings.TrimSpace(terms), ".")
		if !isQuote && doc.DueDate != nil && strings.TrimSpace(*doc.DueDate) != "" {
			line += ", au plus tard le " + formatDate(*doc.DueDate)
		}
		line = ensurePeriod(line)
		if !isQuote {
			if iban := strings.TrimSpace(issuer.IBAN); iban != "" {
				line += " IBAN : " + iban + "."
			}
		}
		out = append(out, line)
	}
	if !isQuote {
		if s := strings.TrimSpace(mentions.LatePenalty); s != "" {
			out = append(out, s)
		}
		if s := strings.TrimSpace(mentions.FlatIndemnity40EUR); s != "" {
			out = append(out, s)
		}
	} else if s := strings.TrimSpace(mentions.QuoteValidityNote); s != "" {
		out = append(out, s)
	}
	if s := strings.TrimSpace(doc.LegalMentions); s != "" {
		out = append(out, s)
	}
	return out
}

func ensurePeriod(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") || strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "…") {
		return s
	}
	return s + "."
}

func displayTitle(kind string) string {
	switch kind {
	case "quote":
		return "DEVIS"
	case "invoice":
		return "FACTURE"
	case "deposit_invoice":
		return "FACTURE D\u2019ACOMPTE"
	case "credit_note":
		return "AVOIR"
	default:
		return "DOCUMENT"
	}
}

func finalLabel(kind string) string {
	switch kind {
	case "invoice":
		return "Net à payer"
	case "deposit_invoice":
		return "Acompte à payer"
	case "credit_note":
		return "Montant de l\u2019avoir"
	default:
		return "Total TTC"
	}
}

func runningHeader(doc biz.Document) string {
	t := displayTitle(doc.Kind)
	if doc.Number != nil && strings.TrimSpace(*doc.Number) != "" {
		return t + "  ·  N° " + strings.TrimSpace(*doc.Number)
	}
	return t
}

func footerLabel(issuer biz.IssuerProfile) string {
	name := strings.TrimSpace(issuer.LegalName)
	if name == "" {
		name = "Devis Generator : VIBED"
	}
	if siret := strings.TrimSpace(issuer.SIRET); siret != "" {
		return name + "  ·  SIRET " + siret
	}
	return name
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func vatBuckets(doc biz.Document) []billing.TaxBucket {
	if len(doc.Lines) == 0 {
		return nil
	}
	in := make([]billing.LineInput, len(doc.Lines))
	for i, l := range doc.Lines {
		in[i] = billing.LineInput{
			Description: l.Description, QuantityMilli: l.QuantityMilli,
			UnitPriceCents: l.UnitPriceCents, DiscountBps: l.DiscountBps, TaxRateBps: l.TaxRateBps,
		}
	}
	res, err := billing.Compute(doc.VATRegime, in)
	if err != nil {
		return nil
	}
	return res.Buckets
}
