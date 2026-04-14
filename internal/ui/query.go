package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/iRootPro/rdr/internal/db"
)

// queryAtomKind enumerates the supported filter primitives.
type queryAtomKind int

const (
	atomFreeWord    queryAtomKind = iota // bare word: matches title OR feed name
	atomStatusRead                       // read / unread
	atomStatusStar                       // starred / unstarred
	atomField                            // title:, feed:, description:
	atomTimeNewer                        // newer than a duration relative to now
	atomTimeOlder                        // older than a duration relative to now
	atomTimeBetween                      // inclusive range (used by today/yesterday)
)

// queryAtom is one filter term parsed from the query string.
type queryAtom struct {
	Kind   queryAtomKind
	Negate bool

	// For atomField / atomFreeWord:
	Field string // "title", "feed", "description", "" for free word
	Value string // lowercased search substring

	// For status:
	StatusValue bool // true=read/starred, false=unread/unstarred

	// For time:
	Since time.Time // used by atomTimeNewer and lower bound of atomTimeBetween
	Until time.Time // used by atomTimeOlder and upper bound (exclusive) of atomTimeBetween
}

// ParseQuery turns a user input string into a list of atoms. All atoms are
// combined with implicit AND. Unknown qualifiers become free words.
func ParseQuery(input string) ([]queryAtom, error) {
	tokens := tokenize(input)
	out := make([]queryAtom, 0, len(tokens))
	now := time.Now().UTC()
	for _, tok := range tokens {
		negate := false
		if strings.HasPrefix(tok, "~") {
			negate = true
			tok = tok[1:]
		}
		if tok == "" {
			continue
		}
		atom, err := parseToken(tok, now)
		if err != nil {
			return nil, err
		}
		atom.Negate = negate
		out = append(out, atom)
	}
	return out, nil
}

// tokenize splits on whitespace. Quoted substrings are NOT supported in MVP —
// users who need phrases can use underscores or rely on substring matching.
func tokenize(s string) []string {
	return strings.Fields(s)
}

func parseToken(tok string, now time.Time) (queryAtom, error) {
	lower := strings.ToLower(tok)

	// Status keywords
	switch lower {
	case "unread":
		return queryAtom{Kind: atomStatusRead, StatusValue: false}, nil
	case "read":
		return queryAtom{Kind: atomStatusRead, StatusValue: true}, nil
	case "starred":
		return queryAtom{Kind: atomStatusStar, StatusValue: true}, nil
	case "unstarred":
		return queryAtom{Kind: atomStatusStar, StatusValue: false}, nil
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return queryAtom{
			Kind:  atomTimeBetween,
			Since: start,
			Until: start.Add(24 * time.Hour),
		}, nil
	case "yesterday":
		start := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)
		return queryAtom{
			Kind:  atomTimeBetween,
			Since: start,
			Until: start.Add(24 * time.Hour),
		}, nil
	}

	// Qualified atoms: "field:value"
	if idx := strings.Index(tok, ":"); idx > 0 {
		field := strings.ToLower(tok[:idx])
		value := tok[idx+1:]
		switch field {
		case "title", "feed", "description":
			if value == "" {
				return queryAtom{}, fmt.Errorf("%s: needs a value", field)
			}
			return queryAtom{
				Kind:  atomField,
				Field: field,
				Value: strings.ToLower(value),
			}, nil
		case "newer":
			d, err := parseDuration(value)
			if err != nil {
				return queryAtom{}, fmt.Errorf("newer: %w", err)
			}
			return queryAtom{Kind: atomTimeNewer, Since: now.Add(-d)}, nil
		case "older":
			d, err := parseDuration(value)
			if err != nil {
				return queryAtom{}, fmt.Errorf("older: %w", err)
			}
			return queryAtom{Kind: atomTimeOlder, Until: now.Add(-d)}, nil
		}
		// Unknown qualifier → fall through to free word with the whole token.
	}

	// Free word: substring over title OR feed name.
	return queryAtom{Kind: atomFreeWord, Value: strings.ToLower(tok)}, nil
}

// parseDuration handles compact forms like "1d", "2w", "3h", "45m", "6mo" (month),
// and "1y" (year). Pure numbers without suffix default to days.
func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	// Split trailing non-digit suffix.
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("duration must start with a number: %q", s)
	}
	n, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, err
	}
	unit := strings.ToLower(s[i:])
	switch unit {
	case "", "d", "day", "days":
		return time.Duration(n) * 24 * time.Hour, nil
	case "w", "week", "weeks":
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case "mo", "month", "months":
		return time.Duration(n) * 30 * 24 * time.Hour, nil
	case "y", "year", "years":
		return time.Duration(n) * 365 * 24 * time.Hour, nil
	case "h", "hour", "hours":
		return time.Duration(n) * time.Hour, nil
	case "m", "min", "minute", "minutes":
		return time.Duration(n) * time.Minute, nil
	case "s", "sec", "second", "seconds":
		return time.Duration(n) * time.Second, nil
	}
	return 0, fmt.Errorf("unknown duration unit %q", unit)
}

// EvalQuery reports whether an item matches all atoms. Empty atom list → true.
func EvalQuery(atoms []queryAtom, it db.SearchItem) bool {
	for _, a := range atoms {
		pass := matchAtom(a, it)
		if a.Negate {
			pass = !pass
		}
		if !pass {
			return false
		}
	}
	return true
}

func matchAtom(a queryAtom, it db.SearchItem) bool {
	switch a.Kind {
	case atomFreeWord:
		return substrFold(it.Title, a.Value) || substrFold(it.FeedName, a.Value)

	case atomStatusRead:
		// StatusValue true = read, false = unread
		isRead := it.ReadAt != nil
		return isRead == a.StatusValue

	case atomStatusStar:
		isStarred := it.StarredAt != nil
		return isStarred == a.StatusValue

	case atomField:
		switch a.Field {
		case "title":
			return substrFold(it.Title, a.Value)
		case "feed":
			return substrFold(it.FeedName, a.Value)
		case "description":
			return substrFold(it.Description, a.Value)
		}
		return false

	case atomTimeNewer:
		return !it.PublishedAt.Before(a.Since)

	case atomTimeOlder:
		return it.PublishedAt.Before(a.Until)

	case atomTimeBetween:
		return !it.PublishedAt.Before(a.Since) && it.PublishedAt.Before(a.Until)
	}
	return false
}

// substrFold does case-insensitive substring match without allocating when
// both sides share length.
func substrFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), needle)
}
