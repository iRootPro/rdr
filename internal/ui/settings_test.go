package ui

import (
	"testing"

	"github.com/iRootPro/rdr/internal/db"
)

func TestSettingsFeedDisplayNavigationFollowsRenderedGroups(t *testing.T) {
	feeds := []db.Feed{
		{ID: 1, Name: "A1", Category: "A"},
		{ID: 2, Name: "B1", Category: "B"},
		{ID: 3, Name: "A2", Category: "A"},
		{ID: 4, Name: "C1", Category: "C"},
	}
	idxs := settingsFeedDisplayIndices(feeds)
	want := []int{0, 2, 1, 3}
	if len(idxs) != len(want) {
		t.Fatalf("idxs len = %d, want %d (%v)", len(idxs), len(want), idxs)
	}
	for i := range want {
		if idxs[i] != want[i] {
			t.Fatalf("idxs = %v, want %v", idxs, want)
		}
	}
	if got := moveSettingsFeedSelection(idxs, 0, 1); got != 2 {
		t.Fatalf("down from rendered first = %d, want 2", got)
	}
	if got := moveSettingsFeedSelection(idxs, 2, 1); got != 1 {
		t.Fatalf("down from rendered second = %d, want 1", got)
	}
	if got := moveSettingsFeedSelection(idxs, 1, 1); got != 3 {
		t.Fatalf("down from rendered third = %d, want 3", got)
	}
}
