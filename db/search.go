package db

import (
	"fmt"
	"regexp"
	"strings"
)

const maxSearchResults = 25

// Double quotes group a phrase. Apostrophes are ordinary word characters.
var searchTerms = regexp.MustCompile(`"([^"]*)"|([^\s"]+)`)

func splitSpacesOrQuoted(text string) []string {
	var terms []string
	for _, match := range searchTerms.FindAllStringSubmatch(text, -1) {
		term := match[1]
		if term == "" {
			term = match[2]
		}
		if term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}

func (d LiteDB) QueryNotesByText(text string) ([]Note, error) {
	return d.queryNotes(text, nil)
}

func (d LiteDB) QueryNotesByTextAndTags(text string, tags []string) ([]Note, error) {
	return d.queryNotes(text, tags)
}

func (d LiteDB) QueryTagsByTitle(text string) ([]Note, error) {
	return d.Query(TAGS_BY_TITLE, text)
}

func (d LiteDB) queryNotes(text string, tags []string) ([]Note, error) {
	query, args := buildSearchQuery(text, tags)
	return d.Query(query, args...)
}

// One statement filters every term and tag, ranks all matching notes, and only
// then applies the result cap. EXISTS avoids multiplying rows by attachments
// and tags, and permits different terms to match different note fields/files.
func buildSearchQuery(text string, tags []string) (string, []interface{}) {
	var args []interface{}
	bind := func(value string) string {
		args = append(args, value)
		return fmt.Sprintf("?%d", len(args))
	}
	terms := splitSpacesOrQuoted(strings.ToLower(text))
	predicates := []string{"note.ZARCHIVED = 0", "note.ZTRASHED = 0", "note.ZTEXT IS NOT NULL"}
	if strings.TrimSpace(text) != "" && len(terms) == 0 {
		// Empty quoted phrases do not turn into a search for every note.
		predicates = append(predicates, "0")
	}
	// Deduplicate filtering work without changing phrase/order ranking.
	seenTerms := map[string]bool{}
	for _, term := range terms {
		if seenTerms[term] {
			continue
		}
		seenTerms[term] = true
		p := bind("%" + term + "%")
		// Preserve the workflow's existing SQL LIKE wildcard behavior for % and _.
		predicates = append(predicates, fmt.Sprintf(`(
   utflower(note.ZTITLE) LIKE %s OR
   utflower(note.ZTEXT) LIKE %s OR
   EXISTS (SELECT 1 FROM ZSFNOTEFILE attachment WHERE attachment.ZNOTE = note.Z_PK
     AND attachment.ZSEARCHTEXT LIKE %s)
  )`, p, p, p))
	}
	seenTags := map[string]bool{}
	for _, tag := range tags {
		tag = strings.ToLower(RemoveTagHashes(tag))
		if seenTags[tag] {
			continue
		}
		seenTags[tag] = true
		predicates = append(predicates, fmt.Sprintf(`EXISTS (
   SELECT 1 FROM Z_5TAGS nt JOIN ZSFNOTETAG tag ON tag.Z_PK = nt.Z_13TAGS
   WHERE nt.Z_5NOTES = note.Z_PK AND utflower(tag.ZTITLE) = %s
  )`, bind(tag)))
	}
	var order []string
	if len(terms) > 0 {
		// Title phrase, title terms in order, all title terms, then recency.
		order = append(order, "(instr(utflower(note.ZTITLE), "+bind(strings.Join(terms, " "))+") > 0) DESC")
		escapeLike := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
		escaped := make([]string, len(terms))
		var allTitle []string
		for i, term := range terms {
			escaped[i] = escapeLike.Replace(term)
		}
		order = append(order, "(utflower(note.ZTITLE) LIKE "+bind("%"+strings.Join(escaped, "%")+"%")+` ESCAPE '\') DESC`)
		for _, term := range terms {
			allTitle = append(allTitle, "instr(utflower(note.ZTITLE), "+bind(term)+") > 0")
		}
		order = append(order, "("+strings.Join(allTitle, " AND ")+") DESC")
	}
	order = append(order, "note.ZMODIFICATIONDATE DESC", "note.ZUNIQUEIDENTIFIER ASC")
	// Ranking deliberately uses the original term sequence, including repeats.
	selectNotes := `SELECT note.ZUNIQUEIDENTIFIER, note.ZTITLE,
  (SELECT GROUP_CONCAT(DISTINCT tag.ZTITLE)
   FROM Z_5TAGS nt JOIN ZSFNOTETAG tag ON tag.Z_PK = nt.Z_13TAGS
   WHERE nt.Z_5NOTES = note.Z_PK) AS TAGS`
	fromWhere := ` FROM ZSFNOTE note WHERE ` + strings.Join(predicates, " AND ")
	ordering := ` ORDER BY ` + strings.Join(order, ", ")
	limit := fmt.Sprintf(" LIMIT %d", maxSearchResults)
	if len(terms) == 0 {
		// For tag-only searches, fetch display tags only for the final results.
		// Keep every filtering condition inside the capped query, and explicitly
		// preserve result order outside it. Text searches keep their simpler plan.
		ranked := `SELECT note.Z_PK, note.ZUNIQUEIDENTIFIER, note.ZTITLE, note.ZMODIFICATIONDATE` + fromWhere + ordering + limit
		return selectNotes + ` FROM (` + ranked + `) note` + ordering, args
	}
	return selectNotes + fromWhere + ordering + limit, args
}
