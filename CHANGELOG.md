# Changelog

## 1.0.0

### Added

- A local SQLite archive and embedded HTMX browser in one Go binary.
- Atomic imports that validate exports before replacing the current archive.
- Chat search, chronological conversation history, older-history loading, and
  links from search results to surrounding messages.
- Images, videos, audio notes, and downloads served from the original export.
- Friends with category filters and links to their chat conversations.
- Snap history with conversation navigation, incoming/outgoing bubbles, search,
  surrounding-snap links, and older-history loading.
- Call history with direction filters and literal text search.
- Account, device, login and profile details, aligned history tables, and compact
  expandable profile collections with entry counts.
- System, Light, and Dark themes, mobile layouts, and native links/forms.
- Optional TOML configuration through urfave/cli v3 and its TOML value sources,
  with explicit CLI flags taking precedence over the file and built-in defaults.
- Configurable HTTP and graceful-shutdown timeouts.
- Browser setup when no archive exists, with an export path field and server-side
  folder browser, background import progress, and automatic navigation to the app.
- Archive controls to select another export or rebuild from the saved source
  without restarting. Failed operations preserve the working archive and media.
- Recovery through folder reselection when a saved export has moved.
- Release archives for Linux, macOS, and Windows on amd64 and arm64, with an
  example configuration, documentation, licenses, and SHA-256 checksums.

### Changed

- The CLI now uses `import` and `serve`. The pre-1.0 parsing commands and flags
  have been replaced; see the README for current usage.
- Imports represent one complete backup. A new import replaces previous data;
  omitted sections of programmatic partial imports are cleared.
- Attachments remain in the original export instead of being copied into SQLite.

### Fixed

- Millisecond timestamps mislabeled as microseconds, including existing archives.
- In-memory databases losing their migrated schema.
- Regression coverage for archive replacement, rollback of data and search
  indexes, profile/friend round trips, configuration, and navigation.
- Tests no longer require a private Snapchat export.

### Current scope

- Eight supported JSON files are imported. Additional JSON files are validated
  but not imported; parser sections still represented by TODO types are omitted.
- Snap history contains metadata rather than attachment IDs.
- Export directories must remain at their recorded location for media playback.
