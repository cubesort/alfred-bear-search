package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mattn/go-sqlite3"
)

func init() {
	sql.Register("bear_fixture", &sqlite3.SQLiteDriver{ConnectHook: func(c *sqlite3.SQLiteConn) error {
		return c.RegisterFunc("utflower", strings.ToLower, true)
	}})
}

func fixture(t testing.TB) LiteDB {
	t.Helper()
	raw, err := sql.Open("bear_fixture", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	t.Cleanup(func() { raw.Close() })
	_, err = raw.Exec(`
 CREATE TABLE ZSFNOTE (Z_PK INTEGER PRIMARY KEY, ZUNIQUEIDENTIFIER TEXT, ZTITLE TEXT,
 ZTEXT TEXT, ZMODIFICATIONDATE REAL, ZARCHIVED INTEGER DEFAULT 0, ZTRASHED INTEGER DEFAULT 0);
 CREATE TABLE ZSFNOTETAG (Z_PK INTEGER PRIMARY KEY, ZTITLE TEXT, ZMODIFICATIONDATE REAL);
 CREATE TABLE Z_5TAGS (Z_5NOTES INTEGER, Z_13TAGS INTEGER);
 CREATE TABLE ZSFNOTEFILE (ZNOTE INTEGER, ZSEARCHTEXT TEXT);
 CREATE INDEX note_tag ON Z_5TAGS(Z_5NOTES);
 CREATE INDEX note_file ON ZSFNOTEFILE(ZNOTE);`)
	if err != nil {
		t.Fatal(err)
	}
	return LiteDB{raw}
}

func addNote(t testing.TB, d LiteDB, id, title, body string, modified int, tags ...string) int64 {
	t.Helper()
	r, err := d.db.Exec(`INSERT INTO ZSFNOTE (ZUNIQUEIDENTIFIER,ZTITLE,ZTEXT,ZMODIFICATIONDATE) VALUES (?,?,?,?)`, id, title, body, modified)
	if err != nil {
		t.Fatal(err)
	}
	pk, _ := r.LastInsertId()
	for _, tag := range tags {
		r, err = d.db.Exec(`INSERT INTO ZSFNOTETAG (ZTITLE,ZMODIFICATIONDATE) VALUES (?,?)`, tag, modified)
		if err != nil {
			t.Fatal(err)
		}
		tagPK, _ := r.LastInsertId()
		if _, err = d.db.Exec(`INSERT INTO Z_5TAGS VALUES (?,?)`, pk, tagPK); err != nil {
			t.Fatal(err)
		}
	}
	return pk
}

func noteIDs(rows []Note) []string {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r[NoteIDKey])
	}
	return ids
}

func expectIDs(t *testing.T, rows []Note, err error, ids ...string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(noteIDs(rows), ids) {
		t.Fatalf("got %v, want %v", noteIDs(rows), ids)
	}
}

func TestSearchBeforeLimit(t *testing.T) {
	for _, tagged := range []bool{false, true} {
		t.Run(fmt.Sprint(tagged), func(t *testing.T) {
			d := fixture(t)
			addNote(t, d, "target", "Older reference", "apple and then banana", 0, "work")
			for i := 0; i < 200; i++ {
				for _, w := range []string{"apple", "banana"} {
					addNote(t, d, fmt.Sprintf("%s-%d", w, i), "Reference", w, i+1, "work")
				}
			}
			var rows []Note
			var err error
			if tagged {
				rows, err = d.QueryNotesByTextAndTags("apple banana", []string{"#work"})
			} else {
				rows, err = d.QueryNotesByText("apple banana")
			}
			expectIDs(t, rows, err, "target")
		})
	}
}

func TestSearchApostrophes(t *testing.T) {
	d := fixture(t)
	addNote(t, d, "exact", "Reference", "O'Reilly", 1)
	addNote(t, d, "unrelated", "Reference", "O followed by Reilly", 2)
	for _, q := range []string{"O'Reilly", `"O'Reilly"`} {
		t.Run(q, func(t *testing.T) { rows, err := d.QueryNotesByText(q); expectIDs(t, rows, err, "exact") })
	}
}

func TestQuotedTagsAndConjunction(t *testing.T) {
	d := fixture(t)
	addNote(t, d, "both", "Reference", "needle", 1, "work's", "project")
	addNote(t, d, "one", "Reference", "needle", 2, "work's")
	addNote(t, d, "other", "Reference", "needle", 3, "project")
	rows, err := d.QueryNotesByTextAndTags("needle", []string{"#work's", "#project"})
	expectIDs(t, rows, err, "both")
	rows, err = d.QueryNotesByTextAndTags("", []string{"#work's", "#project"})
	expectIDs(t, rows, err, "both")
}

func TestRankingAndFinalCap(t *testing.T) {
	d := fixture(t)
	for i := 0; i < 205; i++ {
		addNote(t, d, fmt.Sprint(i), "Reference", "apple banana", i+10)
	}
	addNote(t, d, "full", "apple banana", "", 1)
	addNote(t, d, "ordered", "apple then banana", "", 2)
	addNote(t, d, "unordered", "banana then apple", "", 3)
	rows, err := d.QueryNotesByText("apple banana")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 25 {
		t.Fatalf("got %d results, want 25", len(rows))
	}
	want := []string{"full", "ordered", "unordered", "204"}
	if !reflect.DeepEqual(noteIDs(rows[:4]), want) {
		t.Fatalf("ranking: got %v", noteIDs(rows[:4]))
	}
}

func TestPhrasesAndAttachments(t *testing.T) {
	d := fixture(t)
	pk := addNote(t, d, "attachment", "Reference", "budget", 1, "work")
	for _, s := range []string{"apple banana", "meeting"} {
		if _, err := d.db.Exec(`INSERT INTO ZSFNOTEFILE VALUES (?,?)`, pk, s); err != nil {
			t.Fatal(err)
		}
	}
	addNote(t, d, "split", "Reference", "apple x banana budget meeting", 2, "work")
	addNote(t, d, "wrongtag", "Reference", "apple banana budget meeting", 3, "personal")
	rows, err := d.QueryNotesByTextAndTags(`"apple banana" budget meeting`, []string{"#work"})
	expectIDs(t, rows, err, "attachment")
}

func TestInactiveAndRecency(t *testing.T) {
	d := fixture(t)
	for i, id := range []string{"old", "new", "archived", "trashed"} {
		addNote(t, d, id, "Reference", "needle", i)
	}
	d.db.Exec(`UPDATE ZSFNOTE SET ZARCHIVED=1 WHERE ZUNIQUEIDENTIFIER='archived'`)
	d.db.Exec(`UPDATE ZSFNOTE SET ZTRASHED=1 WHERE ZUNIQUEIDENTIFIER='trashed'`)
	rows, err := d.QueryNotesByText("needle")
	expectIDs(t, rows, err, "new", "old")
}

func TestLikeWildcardsRemainSupported(t *testing.T) {
	d := fixture(t)
	addNote(t, d, "hit", "Reference", "100 things aXb", 1)
	rows, err := d.QueryNotesByText("100% a_b")
	expectIDs(t, rows, err, "hit")
}

func TestTagAutocompleteParameters(t *testing.T) {
	d := fixture(t)
	addNote(t, d, "one", "Reference", "needle", 1, "work's", "work's/project")
	addNote(t, d, "two", "Reference", "needle", 2, "personal")
	rows, err := d.QueryTagsByTitle("work's")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %v, want both apostrophe tags", rows)
	}
	for _, row := range rows {
		if !strings.HasPrefix(row[TitleKey], "work's") {
			t.Fatalf("unexpected tag %v", row)
		}
	}
	rows, err = d.QueryTagsByTitle("' OR 1=1 --")
	if err != nil || len(rows) != 0 {
		t.Fatalf("SQL-shaped input must be literal: rows=%v err=%v", rows, err)
	}
	rows, err = d.QueryNotesByTextAndTags("needle", []string{"#' OR 1=1 --"})
	if err != nil || len(rows) != 0 {
		t.Fatalf("SQL-shaped tag must be literal: rows=%v err=%v", rows, err)
	}
}

func TestRepeatedTermsAndEmptyPhrase(t *testing.T) {
	d := fixture(t)
	addNote(t, d, "one", "Reference", "apple", 1)
	rows, err := d.QueryNotesByText("apple apple")
	expectIDs(t, rows, err, "one")
	rows, err = d.QueryNotesByText(`""`)
	if err != nil || len(rows) != 0 {
		t.Fatalf("empty phrase: rows=%v err=%v", rows, err)
	}
}

func TestReadOnlyOpenAndMissingDatabase(t *testing.T) {
	path := t.TempDir() + "/database.sqlite"
	d, err := NewLiteDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = d.db.Exec("CREATE TABLE fixture (value TEXT)"); err != nil {
		t.Fatal(err)
	}
	d.Close()
	ro, err := NewLiteDB("file:" + path + "?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if _, err = ro.Query("SELECT value FROM fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err = ro.db.Exec("INSERT INTO fixture VALUES ('blocked')"); err == nil {
		t.Fatal("read-only connection accepted a write")
	}
	missing, err := NewLiteDB("file:" + t.TempDir() + "/missing.sqlite?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Close()
	if _, err = missing.Query("SELECT 1"); err == nil {
		t.Fatal("missing database should fail")
	}
}
