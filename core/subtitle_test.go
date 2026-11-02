package core

import (
	"github.com/drgrib/alfred-bear/db"
	"testing"
)

func TestTagsOnlySubtitle(t *testing.T) {
	for _, c := range []struct{ tags, want string }{
		{"work,planning", "#planning #work"},
		{"work", "#work"},
		{"work,workshop", "#work #workshop"},
		{"work,work/project", "#work/project"},
		{"work,work/project,work/project/milestone,workshop", "#work/project/milestone #workshop"},
		{"work,work", "#work"},
		{"work/project,work/project,work", "#work/project"},
		{"my work,my workshop", "#my work# #my workshop#"},
		{"", ""},
	} {
		row := db.Note{db.TitleKey: "Reference", db.NoteIDKey: "note-id", db.TagsKey: c.tags}
		item := RowToItem(row, ParseQuery("budget"))
		if item.Subtitle != c.want {
			t.Fatalf("got %q, want %q", item.Subtitle, c.want)
		}
		if item.Title != "Reference" || item.Arg != "note-id|term=budget" {
			t.Fatal("subtitle changed note action", item)
		}
	}
}
