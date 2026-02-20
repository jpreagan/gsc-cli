package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Printer struct {
	out   io.Writer
	json  bool
	plain bool
	color bool
}

func NewPrinter(out io.Writer, jsonOut, plainOut, color bool) *Printer {
	return &Printer{
		out:   out,
		json:  jsonOut,
		plain: plainOut,
		color: color,
	}
}

func (p *Printer) JSONEnabled() bool  { return p.json }
func (p *Printer) ColorEnabled() bool { return p.color }

func (p *Printer) PrintJSON(v any) error {
	enc := json.NewEncoder(p.out)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	var out io.Writer = p.out
	var buf bytes.Buffer
	if p.color && len(headers) > 0 {
		out = &buf
	}

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		h := make([]string, len(headers))
		for i := range headers {
			h[i] = sanitizeCell(headers[i])
		}
		if _, err := fmt.Fprintln(tw, strings.Join(h, "\t")); err != nil {
			return err
		}
	}
	for _, row := range rows {
		r := make([]string, len(row))
		for i := range row {
			r[i] = sanitizeCell(row[i])
		}
		if _, err := fmt.Fprintln(tw, strings.Join(r, "\t")); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if p.color && len(headers) > 0 {
		return writeHeaderColored(p.out, buf.String())
	}
	return nil
}

func (p *Printer) PrintPlain(rows [][]string) error {
	for _, row := range rows {
		r := make([]string, len(row))
		for i := range row {
			r[i] = sanitizeCell(row[i])
		}
		if _, err := fmt.Fprintln(p.out, strings.Join(r, "\t")); err != nil {
			return err
		}
	}
	return nil
}

func (p *Printer) Print(headers []string, rows [][]string, jsonValue any) error {
	if p.json {
		return p.PrintJSON(jsonValue)
	}
	if p.plain {
		return p.PrintPlain(rows)
	}
	return p.PrintTable(headers, rows)
}

func bold(s string) string {
	return "\x1b[1m" + s + "\x1b[0m"
}

func writeHeaderColored(out io.Writer, table string) error {
	header, tail, hasNewline := strings.Cut(table, "\n")
	if !hasNewline {
		_, err := io.WriteString(out, bold(header))
		return err
	}
	if _, err := io.WriteString(out, bold(header)+"\n"); err != nil {
		return err
	}
	_, err := io.WriteString(out, tail)
	return err
}

func sanitizeCell(s string) string {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '\t' || b == '\n' || b < 0x20 || b == 0x7f {
			var sb strings.Builder
			sb.Grow(len(s))
			sb.WriteString(s[:i])
			for ; i < len(s); i++ {
				b = s[i]
				if b == '\t' || b == '\n' {
					sb.WriteByte(' ')
					continue
				}
				if b < 0x20 || b == 0x7f {
					continue
				}
				sb.WriteByte(b)
			}
			return sb.String()
		}
	}
	return s
}
