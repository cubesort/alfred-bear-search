package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const (
	DbPath    = "~/Library/Group Containers/9K33E3U3T4.net.shinyfrog.bear/Application Data/database.sqlite"
	TitleKey  = "ZTITLE"
	TextKey   = "ZTEXT"
	TagsKey   = "TAGS"
	NoteIDKey = "ZUNIQUEIDENTIFIER"
)

// Keep all list caps tied to the same search limit.
var (
	RECENT_NOTES = fmt.Sprintf(`
SELECT
    note.ZUNIQUEIDENTIFIER,
    note.ZTITLE,
    GROUP_CONCAT(tag.ZTITLE) AS TAGS
FROM
    ZSFNOTE note
LEFT JOIN
    Z_5TAGS nTag ON note.Z_PK = nTag.Z_5NOTES
LEFT JOIN
    ZSFNOTETAG tag ON nTag.Z_13TAGS = tag.Z_PK
WHERE
    note.ZARCHIVED = 0
    AND note.ZTRASHED = 0
GROUP BY
    note.ZUNIQUEIDENTIFIER,
    note.ZTITLE
ORDER BY
    note.ZMODIFICATIONDATE DESC, note.ZUNIQUEIDENTIFIER ASC
LIMIT %d
	`, maxSearchResults)
	TAGS_BY_TITLE = fmt.Sprintf(`
SELECT
    DISTINCT t.ZTITLE
FROM
    ZSFNOTE n
INNER JOIN
    Z_5TAGS nt ON n.Z_PK = nt.Z_5NOTES
INNER JOIN
    ZSFNOTETAG t ON nt.Z_13TAGS = t.Z_PK
WHERE
    n.ZARCHIVED = 0
    AND n.ZTRASHED = 0
    AND UTFLOWER(t.ZTITLE) LIKE UTFLOWER('%%'||$1||'%%')
ORDER BY
    t.ZMODIFICATIONDATE DESC, t.ZTITLE ASC
LIMIT %d

`, maxSearchResults)
)

type Note map[string]string

func Expanduser(path string) string {
	usr, err := user.Current()
	if err != nil {
		return path
	}
	dir := usr.HomeDir
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(dir, path[2:])
	}
	return path
}

type LiteDB struct {
	db *sql.DB
}

var registerDriver sync.Once

func NewLiteDB(path string) (LiteDB, error) {
	registerDriver.Do(func() {
		sql.Register("sqlite3_custom", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("utflower", utfLower, true)
			},
		})
	})
	db, err := sql.Open("sqlite3_custom", path)
	litedb := LiteDB{db}
	return litedb, err
}

func utfLower(s string) string {
	return strings.ToLower(s)
}

func NewBearDB() (LiteDB, error) {
	path := Expanduser(DbPath)
	// Bear owns this database; never create or modify it.
	uri := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}
	litedb, err := NewLiteDB(uri.String())
	return litedb, err
}

func (litedb LiteDB) Query(q string, args ...interface{}) ([]Note, error) {
	results := []Note{}
	rows, err := litedb.db.Query(q, args...)
	if err != nil {
		return results, errors.WithStack(err)
	}

	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return results, errors.WithStack(err)
	}

	columns := make([]interface{}, len(cols))
	columnPointers := make([]interface{}, len(cols))
	for i := range columns {
		columnPointers[i] = &columns[i]
	}
	for rows.Next() {
		m := make(Note, len(cols))
		if err := rows.Scan(columnPointers...); err != nil {
			return results, errors.WithStack(err)
		}
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			s, ok := (*val).(string)
			if ok {
				m[colName] = s
			} else {
				m[colName] = ""
			}
		}
		results = append(results, m)
	}
	err = rows.Close()
	if err != nil {
		return results, errors.WithStack(err)
	}
	err = rows.Err()
	if err != nil {
		return results, errors.WithStack(err)
	}
	return results, errors.WithStack(err)
}

func (litedb LiteDB) Close() error { return litedb.db.Close() }

func RemoveTagHashes(tag string) string {
	tag = strings.TrimPrefix(tag, "#")
	if strings.HasSuffix(tag, "#") {
		tag = tag[:len(tag)-1]
	}
	return tag
}
