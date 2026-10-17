# Embedded archive browser

Start directly from an unzipped Snapchat export; this creates the SQLite archive
and starts the viewer in one command:

```sh
go run ./cmd/snaparser serve --data ./data
# Open http://127.0.0.1:8080

# On subsequent runs, reuse the imported database without importing again:
go run ./cmd/snaparser serve

# Or choose an existing database:
go run ./cmd/snaparser serve --db /path/to/archive.db
```

`--data` requires the unchanged export root containing `json/` and `chat_media/`.
It parses every supported JSON file and validates additional JSON files before
atomically replacing the selected archive using `db.Import`. Missing directories,
missing supported JSON files, or malformed JSON abort without changing an
existing archive. Additional JSON contents are not imported. The absolute export
root is stored in the same transaction as the parsed data. The default database is
`snaparser.db`; `--db` selects another path. Import without starting the viewer
with `go run ./cmd/snaparser import --data ./data --db snaparser.db`.

`--listen` changes the default `127.0.0.1:8080` address. The CLI also loads
optional `snaparser.toml` configuration, or a file selected by `--config`.
Explicit flags override TOML. See the root README and `snaparser.example.toml`
for paths and server timeout settings. Without `--data`, the
command requires an existing database file. Opening it applies the existing
database package's migrations. Keep the source export at its imported location;
viewer startup aborts if its recorded media directory cannot be read. Archives
without a recorded source still display chats; reimport from the export root to
enable attachments. An empty archive gets a welcome state. There is
no demo data, and the viewer has no HTTP import endpoint.

To integrate the browser with another server, mount the handler at `/`:

```go
viewer, err := frontend.New(database) // *db.DB satisfies frontend.Store
if err != nil {
    return err
}
mux.Handle("/", viewer)
```

The caller owns the database and HTTP server lifecycle. The existing database
package remains the only data access layer. The frontend doesn't access its
connection, modify its schema, or perform imports. Its Store interface also lets
another server supply its own adapter for chats, snaps, friends, calls, account
history, account/profile details, and the archive source.

The handler serves complete Go `html/template` documents for normal requests
and `workspace` fragments for HTMX. Search replaces only `#archive`, preserving
input focus while typing. Links and forms also work without JavaScript. URLs
preserve search, selected conversation, and separate message/conversation pages;
browser history restoration receives a complete document.

Search treats input as literal word prefixes combined with AND, searching the
existing chats FTS index (message content, sender, and conversation title).
Quoted input is escaped rather than interpreted as FTS operators. Each request
fetches at most 41 chats and 26 conversation summaries to detect the next page.
Conversations render oldest-to-newest and open at the latest message. Scrolling
to the top automatically prepends earlier history without replacing the current
messages. Load older messages also works manually; failed loads offer a Retry
button. Latest messages returns to the bottom. The message pane
scrolls independently of the fixed page. Native links page through older and
newer history when HTMX is unavailable. Search results remain ranked by relevance
and use ordinary pagination. Times are UTC.
Click a search result's text or its View in conversation link to open the full
conversation around that message. The viewer clears the search, loads a window
of surrounding messages, scrolls to the result, and highlights its bubble. Older
and newer history remain available. Context links include a message ID and
fragment so they also work without JavaScript, on refresh, and in a new tab.
Browser Back returns to the search results.
Exports with milliseconds mislabeled as microseconds are normalized on import;
existing databases are corrected automatically when opened, without reimporting.
Messages use pale yellow incoming bubbles on the left and light blue outgoing
bubbles on the right, with sender names, timestamps, and saved status.
The desktop conversation sidebar becomes a native selector on phones. The
header's Theme selector offers System, Light, and Dark. CSS applies the choice
immediately; HTMX persists it in a one-year cookie without replacing the page
or losing the active search or scroll position. With JavaScript disabled, use
the Apply button. System follows the device's appearance setting. All UI palette values live
in the two theme blocks at the top of `assets/app.css`; component rules only
reference those shared color variables.

After JSON import, the viewer builds one media filename index at startup and
resolves attachments for each displayed chat using its UTC `Created` date and
pipe-separated `Media IDs`. Matching requires the exact `YYYY-MM-DD_<ID>` stem;
the actual filename supplies the extension. Images appear inline, video and audio
use native controls, and other types get download links. Captions and multiple
attachments are supported. Snapchat `NOTE` messages use audio controls even when
their files have an `.mp4` extension. Missing files retain the unavailable state.
`/media/{filename}` reads only regular files inside the recorded `chat_media`
directory, supports HEAD and byte ranges for playback, and disables caching.
Restart the viewer after importing another export. Media files are not embedded
or copied to the database.

All application assets use `go:embed`: plain CSS, an SVG icon, and the unmodified
[HTMX 2.0.11 distribution](https://htmx.org/docs/#installing), with its license
in `assets/HTMX-LICENSE`. No custom JavaScript, inline scripts, CDN requests,
remote fonts, Node, or frontend build step. Archive responses use `no-store`,
and HTMX history snapshots are disabled to keep messages out of localStorage.

```sh
go test ./frontend ./cmd/snaparser
go build ./cmd/snaparser
```

## Other archive views

The navigation inside `#workspace` provides Chats (`/`), Friends (`/friends`),
Snap history (`/snaps`), Calls (`/calls`), and Account (`/account`). All routes
return complete documents for native navigation and history restoration, or
workspace fragments for HTMX. They share themes, cache headers, and error states.
Friends and Calls use scrollable tables on small screens and fetch at most 41
rows to display 40 records and detect a next page. Snap history uses the chat
layout and history-loading behavior described above. Search/filter changes reset the
record offset; pagination preserves the active search and filter.

Friends supports literal word-prefix search over usernames and display names,
and a category selector for friends, sent requests, blocked, deleted, hidden
suggestions, ignored, and pending requests. Use `q`, `filter`, and `offset` in
URLs. Search updates only `#records`, preserving the search input and focus.
Usernames and display names link to the matching chat conversation using the
exact exported username, across all relationship categories. These links clear
the friends search, category, and page. Friends without an imported conversation
show “No chat history” instead of a link.

Snap history uses the same conversation sidebar, mobile selector, date dividers,
incoming/outgoing bubbles, and independent message scrolling as Chats. Its sidebar
includes conversations containing snaps, ordered by their latest snap activity,
with snap counts. It opens on the newest window, rendered oldest to newest;
scrolling up or selecting Load older snaps prepends earlier history. Native
navigation provides newer/older pages and Latest snaps.

Snap search matches the sender and conversation title, with relevance ordering
and pagination. Search results link to surrounding snaps, clearing the search
and highlighting the selected snap. Search, conversation selection, context
links, and history navigation stay under `/snaps`. The `conversation`, `offset`,
`conversations_offset`, and `message` parameters work as in Chats. Headers link
between the same conversation's Chats and Snap history views. Bubbles display
sender, direction, media type, and UTC time. These are metadata records; the
export's snap history has no media IDs for resolving attachments.

Calls supports `filter=incoming` or `filter=outgoing`, with a literal substring
search across the exported date, type, result, city, country, and network. SQL
wildcards and FTS operators have no special meaning here. Results use descending
exported date strings, then descending ID. Duration is displayed in seconds;
participant counts are shown because the imported call records do not include
participant names.

Account shows all currently stored account/device/profile fields, device and
login histories, engagement, viewed channels, advertising interactions, content
categories, time spent, and web/app interactions. Account overview cards share a
two-column grid (one column on phones); history tables use full-width sections
with fixed column layouts and sticky headers. Profile collections are collapsed
by default with entry counts. Opening a collection reveals its complete contents
in a bounded, keyboard-scrollable area, with multi-column lists on wider screens.
Account history pagination sits inside its own section. Account changes are paged in
import order, with the event type, date, value, and status. Profile URLs and
image references appear as text; viewing the account page does not fetch remote
profile resources. Sections represented by parser TODO placeholders remain
unsupported.
