# Bear for Alfred

Search and open Bear notes from Alfred. Find notes by title, body text, tags, or
attachment text already indexed by Bear, with results shown directly in Alfred.

A search-focused fork of [drgrib/alfred-bear](https://github.com/drgrib/alfred-bear),
originally created by Chris Redford.

## Requirements

- macOS with Bear installed and a local note database.
- [Alfred with the Powerpack](https://www.alfredapp.com/powerpack/), which enables workflows.
- To build from source: [Go](https://go.dev/dl/) 1.21 or newer and Apple's Command Line Tools.

## Installation

If you already have a built `.alfredworkflow` file, double-click it to import it
into Alfred. Go and the Command Line Tools are needed only when building.

To build from source:

1. Clone or download this repository.
2. Install Go and, if needed, Apple's Command Line Tools:

   ```sh
   xcode-select --install
   ```

3. From the repository directory, run:

   ```sh
   bash build-macos.command
   ```

4. Open `dist/Bear.alfredworkflow` to import it into Alfred.

The script runs the tests, builds the search executable for the installed Go
toolchain's Mac architecture, and packages the workflow. Dependencies are vendored,
so the build does not need to download Go modules.

## Usage

Open Alfred and type `b`, followed by a space and your search. Leave the query
empty to see recent notes.

| Action | Result |
| --- | --- |
| Return | Open the selected note in Bear's main window |
| Command-Return | Open the selected note in a new window |
| Option-Return | Search the query in Bear |

Change the keyword or configure the Hotkey object in Alfred's workflow editor.

### Search examples

| Input | Finds |
| --- | --- |
| `b budget` | Notes containing `budget` |
| `b budget launch` | Notes containing both terms |
| `b "project launch"` | Notes containing the phrase `project launch` |
| `b budget #work` | Notes containing `budget` with the `work` tag |
| `b #work #planning` | Notes with both tags |
| `b #meeting notes#` | Notes with the multiword tag `meeting notes` |

Typing `#` starts tag autocomplete. Add a space after the final tag to leave
autocomplete and display matching notes. Single-word tags accept either `#work`
or `#work#`; multiword tags need the closing `#`.

### Matching and ranking

All search terms and tags must match. Different terms can occur in different
fields of the same note, including its indexed attachments. Archived and trashed
notes are excluded.

Results are ranked by:

1. The full search phrase appearing in the title.
2. All search terms appearing in the title in query order.
3. All search terms appearing anywhere in the title.
4. Most recently modified notes.

Matching and ranking happen before the **25-result limit**. Recent notes and tag
autocomplete are also limited to 25 items. Alfred displays the workflow's results
without applying an additional text filter.

Subtitles show tags only. Duplicate tags appear once, and parent tags are hidden
when a child is present: `#work` and `#work/project` display as `#work/project`.
Unrelated tags such as `#work` and `#workshop` both remain visible.

### Search details

- Apostrophes are literal characters, so `O'Reilly` remains one search term.
- `%` and `_` retain SQL wildcard behavior: `%` matches any sequence of characters,
  and `_` matches one character.
- Bear-specific operators such as `@todo` are not interpreted by the local search.
  Use Option-Return to run a query in Bear.
- Search in Bear passes the text query and the first selected tag. Local searches
  apply every selected tag.
- Attachment searches use text already indexed by Bear; the workflow does not
  extract attachment text or perform OCR itself.

## Customization and updates

Alfred objects, connections, keywords, hotkeys, and workflow metadata are stored
in `workflow-template/info.plist`. The icon is `workflow-template/icon.png`.

After editing the installed workflow in Alfred, export it and extract its
`info.plist` into `workflow-template/info.plist` to preserve those settings in
future builds. An `.alfredworkflow` file is a ZIP archive.

To update only the search implementation while keeping your installed workflow
settings, build the workflow and replace `cmd/search/search` in the installed
workflow folder with the executable from the new archive. Preserve its executable
permission.

## Troubleshooting

### The build cannot find `errno.h`, `stdlib.h`, or `pthread.h`

The SQLite driver uses cgo and needs Apple's compiler and macOS SDK. The build
script selects the SDK explicitly and checks its headers before compiling.

Inspect the active developer tools:

```sh
xcode-select -p
xcrun --sdk macosx --show-sdk-path
```

Install or update the Command Line Tools, or select a complete Xcode installation,
then rerun `bash build-macos.command`.

### Alfred shows “Unable to search Bear notes”

Open Bear and confirm that its notes are available locally. Then open Alfred's
workflow debugger and repeat the search to see the database error.

The workflow reads Bear's database at:

```text
~/Library/Group Containers/9K33E3U3T4.net.shinyfrog.bear/Application Data/database.sqlite
```

Search opens this database read-only. The workflow depends on Bear's internal
database schema; a Bear update that changes the schema may require a code update.

## Development

| Path | Purpose |
| --- | --- |
| `cmd/search/` | Search executable and error output |
| `core/` | Query parsing, tag formatting, and Alfred results |
| `db/` | SQLite access, matching, ranking, tests, and benchmarks |
| `workflow-template/` | Alfred configuration and assets |
| `vendor/` | Go dependencies and their licenses |
| `build-macos.command` | Native Mac build and workflow packaging |

Run the tests and static checks:

```sh
go test -mod=vendor ./... -count=1
go vet -mod=vendor ./...
```

Tests use synthetic SQLite databases and do not require access to personal Bear
notes. They cover matching before the result limit, ranking, tags, phrases,
attachments, URL encoding, read-only access, and error handling.

Run the search benchmark:

```sh
go test -mod=vendor ./db -run '^$' -bench BenchmarkSearchPatterns -benchmem -benchtime=300ms -count=3
```

Use `-bench BenchmarkResultCap` to compare result limits. Benchmarks use synthetic
data and exclude Alfred, process startup, and real-database disk I/O; measure on
your Mac when evaluating performance changes.

## License

[MIT](LICENSE). Original work copyright © 2024 Chris Redford.
Third-party dependencies retain their respective licenses in `vendor/`.
