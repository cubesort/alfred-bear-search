package db

import (
	"fmt"
	"strings"
	"testing"
)

func searchBenchmarkData(t testing.TB) LiteDB {
	d := fixture(t)
	for i := 0; i < 5000; i++ {
		title := "Reference"
		body := strings.Repeat("ordinary context ", 50) + " apple"
		if i%17 == 0 {
			body += " banana"
		}
		if i%50 == 0 {
			title = "apple banana"
		}
		pk := addNote(t, d, fmt.Sprintf("%05d", i), title, body, i, "work", "project", "work/subtag")
		if i%11 == 0 {
			if _, err := d.db.Exec("INSERT INTO ZSFNOTEFILE VALUES (?,?)", pk, "attachment meeting"); err != nil {
				t.Fatal(err)
			}
		}
	}
	return d
}

var benchCases = []struct {
	name, text string
	tags       []string
}{
	{"common", "apple", nil}, {"multiword", "apple banana", nil},
	{"repeated", "apple apple banana", []string{"#work", "#work", "#project"}},
	{"tag_only", "", []string{"#work"}},
}

func BenchmarkSearchPatterns(b *testing.B) {
	d := searchBenchmarkData(b)
	for _, c := range benchCases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				q, args := buildSearchQuery(c.text, c.tags)
				if _, err := d.Query(q, args...); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
