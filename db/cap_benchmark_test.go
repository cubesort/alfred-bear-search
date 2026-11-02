package db

import (
	"fmt"
	"strings"
	"testing"
)

func TestAllResultListsCappedAt25(t *testing.T) {
	d := fixture(t)
	for i := 0; i < 80; i++ {
		addNote(t, d, fmt.Sprint(i), "Reference", "needle", i, "work", fmt.Sprintf("tag-%02d", i))
	}
	checks := []struct {
		name string
		run  func() ([]Note, error)
	}{
		{"text", func() ([]Note, error) { return d.QueryNotesByText("needle") }},
		{"text and tag", func() ([]Note, error) { return d.QueryNotesByTextAndTags("needle", []string{"#work"}) }},
		{"tag only", func() ([]Note, error) { return d.QueryNotesByTextAndTags("", []string{"#work"}) }},
		{"recent", func() ([]Note, error) { return d.Query(RECENT_NOTES) }},
		{"tag autocomplete", func() ([]Note, error) { return d.QueryTagsByTitle("tag-") }},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			rows, err := c.run()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 25 {
				t.Fatalf("got %d results; want 25", len(rows))
			}
		})
	}
}

// Isolate the cap change: both cases use the same corrected query and data.
// Only the final LIMIT differs. Query construction and JSON output are excluded.
func BenchmarkResultCap(b *testing.B) {
	d := fixture(b)
	for i := 0; i < 5000; i++ {
		body := "background text and ordinary reference details"
		if i%10 == 0 {
			body += " apple"
		}
		if i%17 == 0 {
			body += " banana"
		}
		addNote(b, d, fmt.Sprint(i), "Reference", body, i, "work", "project")
	}
	for _, tagged := range []bool{false, true} {
		var tags []string
		if tagged {
			tags = []string{"#work", "#project"}
		}
		for _, text := range []string{"apple", "apple banana"} {
			query, args := buildSearchQuery(text, tags)
			if !strings.HasSuffix(query, " LIMIT 25") {
				b.Fatal("unexpected final limit")
			}
			for _, cap := range []int{200, 25} {
				sql := strings.TrimSuffix(query, " LIMIT 25") + fmt.Sprintf(" LIMIT %d", cap)
				b.Run(fmt.Sprintf("tagged=%t/%s/cap=%d", tagged, text, cap), func(b *testing.B) {
					rows, err := d.Query(sql, args...)
					if err != nil {
						b.Fatal(err)
					}
					expected := cap
					if text == "apple banana" && cap == 200 {
						expected = 30
					}
					if len(rows) != expected {
						b.Fatalf("got %d rows, want %d", len(rows), expected)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if _, err := d.Query(sql, args...); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		}
	}
}
