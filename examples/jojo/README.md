# JoJo demo export

A fictional Snapchat-style export inspired by *JoJo's Bizarre Adventure*, used
for the screenshots in the main README. The conversations are original demo
dialogue; the account details, activity, and contact information are made up.
Characters from different parts share this deliberately imaginary trip.

The demo files are included in release archives under `examples/jojo/`. From the
extracted release directory:

```sh
./snaparser serve --data ./examples/jojo --db ./jojo-demo.db
# Open http://127.0.0.1:8080 and select stardust.crusaders.
```

On Windows, use `.\snaparser.exe` in place of `./snaparser`. The demo is not
embedded in the executable; a `go install` installation needs these files from a
release archive or source checkout.

From the repository root of a source checkout:

```sh
go run ./cmd/snaparser serve --data ./examples/jojo --db ./jojo-demo.db
# Open http://127.0.0.1:8080 and select stardust.crusaders.
```

Use the separate demo database to keep your own archive. The export includes all
eight supported JSON files, a route-map image, and a downloadable travel checklist.
Keep this directory in place while browsing its attachments. The same folder can
also be selected through the app's Archive page.
