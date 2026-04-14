package feed

import (
	"bytes"
	"strings"
	"testing"
)

const sampleOPML = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>My Feeds</title></head>
  <body>
    <outline text="News" title="News">
      <outline type="rss" text="Hacker News" title="Hacker News"
               xmlUrl="https://news.ycombinator.com/rss" htmlUrl="https://news.ycombinator.com/"/>
      <outline type="rss" text="Lobsters" title="Lobsters"
               xmlUrl="https://lobste.rs/rss"/>
    </outline>
    <outline type="rss" text="Top Level" title="Top Level"
             xmlUrl="https://top.example/rss"/>
  </body>
</opml>`

func TestParseOPML_FlattensNestedCategories(t *testing.T) {
	entries, err := ParseOPML(strings.NewReader(sampleOPML))
	if err != nil {
		t.Fatalf("ParseOPML: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d: %+v", len(entries), entries)
	}
	want := map[string]string{
		"Hacker News": "https://news.ycombinator.com/rss",
		"Lobsters":    "https://lobste.rs/rss",
		"Top Level":   "https://top.example/rss",
	}
	for _, e := range entries {
		if url, ok := want[e.Name]; !ok {
			t.Fatalf("unexpected entry: %+v", e)
		} else if url != e.URL {
			t.Fatalf("url for %q: got %q, want %q", e.Name, e.URL, url)
		}
	}
}

func TestParseOPML_IgnoresEmptyXMLURL(t *testing.T) {
	// Category outlines without xmlUrl should not become entries.
	entries, err := ParseOPML(strings.NewReader(sampleOPML))
	if err != nil {
		t.Fatalf("ParseOPML: %v", err)
	}
	for _, e := range entries {
		if e.Name == "News" {
			t.Fatalf("category 'News' leaked into entries: %+v", e)
		}
	}
}

func TestParseOPML_FallbackNameFromText(t *testing.T) {
	raw := `<?xml version="1.0"?><opml version="2.0"><body>
		<outline text="Only Text" xmlUrl="https://t.example/rss"/>
	</body></opml>`
	entries, err := ParseOPML(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseOPML: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "Only Text" {
		t.Fatalf("want 1 entry with name 'Only Text', got %+v", entries)
	}
}

func TestWriteOPML_RoundTrip(t *testing.T) {
	in := []OPMLEntry{
		{Name: "Alpha", URL: "https://a.example/rss"},
		{Name: "Beta", URL: "https://b.example/rss"},
	}
	var buf bytes.Buffer
	if err := WriteOPML(&buf, "rdr export", in); err != nil {
		t.Fatalf("WriteOPML: %v", err)
	}
	out, err := ParseOPML(&buf)
	if err != nil {
		t.Fatalf("ParseOPML: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("round-trip: want %d, got %d", len(in), len(out))
	}
	for i := range in {
		if out[i].Name != in[i].Name || out[i].URL != in[i].URL {
			t.Fatalf("round-trip mismatch: got %+v, want %+v", out[i], in[i])
		}
	}
}
