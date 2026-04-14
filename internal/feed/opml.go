package feed

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// OPMLEntry is a flat representation of a feed extracted from an OPML file.
type OPMLEntry struct {
	Name string
	URL  string
}

type opmlDoc struct {
	XMLName xml.Name    `xml:"opml"`
	Version string      `xml:"version,attr"`
	Head    opmlHead    `xml:"head"`
	Body    opmlOutline `xml:"body"`
}

type opmlHead struct {
	Title       string `xml:"title,omitempty"`
	DateCreated string `xml:"dateCreated,omitempty"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr,omitempty"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline"`
}

// ParseOPML decodes an OPML document and returns a flat list of feed entries.
// Nested categories are flattened — only outlines with a non-empty xmlUrl are
// emitted. The title/text attribute is preferred for the entry name.
func ParseOPML(r io.Reader) ([]OPMLEntry, error) {
	var doc opmlDoc
	dec := xml.NewDecoder(r)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode opml: %w", err)
	}
	var out []OPMLEntry
	flattenOutlines(&out, doc.Body.Outlines)
	return out, nil
}

func flattenOutlines(out *[]OPMLEntry, outlines []opmlOutline) {
	for _, o := range outlines {
		if url := strings.TrimSpace(o.XMLURL); url != "" {
			name := strings.TrimSpace(o.Title)
			if name == "" {
				name = strings.TrimSpace(o.Text)
			}
			if name == "" {
				name = url
			}
			*out = append(*out, OPMLEntry{Name: name, URL: url})
		}
		if len(o.Outlines) > 0 {
			flattenOutlines(out, o.Outlines)
		}
	}
}

// WriteOPML serializes feeds as an OPML 2.0 document. Each feed becomes a
// top-level outline under <body>; no category nesting is produced.
func WriteOPML(w io.Writer, title string, feeds []OPMLEntry) error {
	doc := opmlDoc{
		Version: "2.0",
		Head: opmlHead{
			Title:       title,
			DateCreated: time.Now().UTC().Format(time.RFC1123),
		},
	}
	for _, f := range feeds {
		doc.Body.Outlines = append(doc.Body.Outlines, opmlOutline{
			Text:   f.Name,
			Title:  f.Name,
			Type:   "rss",
			XMLURL: f.URL,
		})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encode opml: %w", err)
	}
	return enc.Flush()
}
