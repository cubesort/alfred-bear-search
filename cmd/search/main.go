package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/drgrib/alfred"

	"github.com/drgrib/alfred-bear/core"
	"github.com/drgrib/alfred-bear/db"
)

func main() {
	text := ""
	if len(os.Args) > 1 {
		text = os.Args[1]
	}
	if err := runSearch(text); err != nil {
		reportError(err)
	}
	alfred.Run()
}

func reportError(err error) {
	fmt.Fprintf(os.Stderr, "Bear search failed: %+v\n", err)
	alfred.Add(alfred.Item{
		Title:    "Unable to search Bear notes",
		Subtitle: "Check Bear is installed and its database is accessible. See Alfred's debugger for details.",
		Valid:    alfred.Bool(false),
	})
}

func runSearch(text string) error {
	query := core.ParseQuery(text)

	litedb, err := db.NewBearDB()
	if err != nil {
		return err
	}
	defer litedb.Close()

	autocompleted, err := core.Autocomplete(litedb, query)
	if err != nil {
		return err
	}

	if !autocompleted {
		rows, err := core.GetSearchRows(litedb, query)
		if err != nil {
			return err
		}
		core.AddNoteRowsToAlfred(rows, query)
		if len(rows) == 0 {
			if strings.Contains(query.WordString, "@") {
				mainWindowItem, err := core.GetAppSearchItem(query)
				if err != nil {
					return err
				}
				alfred.Add(*mainWindowItem)
			} else {
				alfred.Add(alfred.Item{
					Title: "No matching items found",
					Valid: alfred.Bool(false),
				})
			}
		}
	}

	return nil
}
