package core

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/drgrib/alfred-bear/db"
)

func TestClosedSingleWordTags(t *testing.T) {
	for _, c := range []struct {
		input, words string
		tags         []string
	}{
		{"#work# ", "", []string{"#work#"}},
		{"needle #work#", "needle", []string{"#work#"}},
		{"#work# needle", "needle", []string{"#work#"}},
		{"#work# #project# ", "", []string{"#work#", "#project#"}},
		{"#work needle #project# ", "needle", []string{"#work", "#project#"}},
		{"#multi word# #work# ", "", []string{"#multi word#", "#work#"}},
		{"C# #work# ", "C#", []string{"#work#"}},
	} {
		t.Run(c.input, func(t *testing.T) {
			got := ParseQuery(c.input)
			if got.WordString != c.words || !reflect.DeepEqual(got.Tags, c.tags) {
				t.Fatalf("got words=%q tags=%v; want words=%q tags=%v", got.WordString, got.Tags, c.words, c.tags)
			}
		})
	}
}

func TestSearchCallbackRoundTrip(t *testing.T) {
	for _, c := range []struct{ words, tag string }{
		{"R&D", "R&D"}, {"C++", "C++"}, {"a=b?c#d|e%f", "a=b?c#d|e%f"},
		{"two words", "multi word"}, {"预算 café", "项目/café"}, {"", ""},
	} {
		t.Run(c.words, func(t *testing.T) {
			q := Query{WordString: c.words}
			if c.tag != "" {
				q.Tags = []string{"#" + c.tag + "#"}
			}
			item := RowToItem(db.Note{db.NoteIDKey: "note-id"}, q)
			parts := strings.Split(item.Arg, "|")
			if len(parts) != 2 || parts[0] != "note-id" {
				t.Fatalf("corrupt note action: %q", item.Arg)
			}
			got, err := url.ParseQuery(parts[1])
			if err != nil || got.Get("term") != c.words || got.Get("tag") != c.tag {
				t.Fatalf("callback %q decoded as %v (%v)", parts[1], got, err)
			}
			for key := range got {
				if key != "term" && key != "tag" {
					t.Fatalf("unexpected parameter %q", key)
				}
			}
			app, err := GetAppSearchItem(q)
			if err != nil || app.Arg != parts[1] {
				t.Fatal("app and note callbacks differ")
			}
		})
	}
}
