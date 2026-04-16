package ui

import "strings"

// feedIcon returns a Nerd Font icon for a feed based on URL/name pattern
// matching. Unknown feeds get a generic RSS icon.
func feedIcon(url, name string) string {
	u := strings.ToLower(url)
	n := strings.ToLower(name)
	for _, m := range iconTable {
		for _, p := range m.patterns {
			if strings.Contains(u, p) || strings.Contains(n, p) {
				return m.icon
			}
		}
	}
	return "\U000f046b" // 󰑫 RSS fallback
}

var iconTable = []struct {
	patterns []string
	icon     string
}{
	{[]string{"github.com", "github"}, "\uf09b"},           //  GitHub
	{[]string{"reddit.com", "reddit"}, "\uf281"},           //  Reddit
	{[]string{"habr.com", "habrahabr", "habr"}, "\uf1d4"},  //  Habr
	{[]string{"news.ycombinator", "hacker news"}, "\uf269"}, //  HN
	{[]string{"youtube.com", "youtube"}, "\uf167"},          //  YouTube
	{[]string{"stackoverflow"}, "\uf16c"},                   //  SO
	{[]string{"medium.com"}, "\uf23a"},                      //  Medium
	{[]string{"dev.to"}, "\ue77b"},                          //  Dev.to
	{[]string{"twitter.com", "x.com"}, "\U000f0544"},        // 󰕄 X
	{[]string{"t.me", "telegram"}, "\uf2c6"},                //  Telegram
	{[]string{"lobste.rs", "lobsters"}, "\U000f0320"},       // 󰌠 Lobsters
	{[]string{"go.dev", "golang", "go blog"}, "\ue627"},     //  Go
	{[]string{"opennet"}, "\uf233"},                         //  Opennet
}
