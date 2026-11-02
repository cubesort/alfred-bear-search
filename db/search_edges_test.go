package db

import (
	"reflect"
	"testing"
)

func TestRepeatedFiltersPreserveRanking(t *testing.T) {
	d := fixture(t)
	addNote(t, d, "single", "apple", "", 100, "work")
	addNote(t, d, "ordered", "apple then apple", "", 2, "work")
	addNote(t, d, "phrase", "apple apple", "", 1, "work")
	rows, err := d.QueryNotesByTextAndTags("apple apple", []string{"#work", "#WORK", "#work#"})
	expectIDs(t, rows, err, "phrase", "ordered", "single")
}

func TestStableRecentAndAutocompleteTies(t *testing.T) {
	d := fixture(t)
	addNote(t, d, "z", "Z", "", 10, "tag-z")
	addNote(t, d, "a", "A", "", 10, "tag-a")
	addNote(t, d, "m", "M", "", 10, "tag-m")
	rows, err := d.Query(RECENT_NOTES)
	expectIDs(t, rows, err, "a", "m", "z")
	rows, err = d.QueryTagsByTitle("tag-")
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, row := range rows {
		tags = append(tags, row[TitleKey])
	}
	if !reflect.DeepEqual(tags, []string{"tag-a", "tag-m", "tag-z"}) {
		t.Fatal(tags)
	}
}

func TestQueryScanDoesNotRetainPreviousRow(t *testing.T) {
	d := fixture(t)
	rows, err := d.Query(`SELECT 'first' AS value UNION ALL SELECT NULL UNION ALL SELECT 'last'`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0]["value"] != "first" || rows[1]["value"] != "" || rows[2]["value"] != "last" {
		t.Fatalf("incorrect row values: %v", rows)
	}
}
