package frontend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/vanillaiice/snaparser/db"
	"github.com/vanillaiice/snaparser/parser"
)

// Archive owns a selectable archive, its viewer, and background import jobs.
// Close it after the HTTP server stops accepting requests.
type Archive struct {
	mu        sync.RWMutex
	viewMu    sync.RWMutex
	workers   sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	path      string
	store     *db.DB
	viewer    http.Handler
	shell     *handler
	routes    http.Handler
	source    string
	selection string
	problem   string
	job       *archiveJob
}

type archiveJob struct {
	id      uint64
	running bool
	stage   string
	done    chan struct{}
}

type folder struct {
	Name string
	URL  string
}

type setupPage struct {
	CurrentDirectory string
	Directory        string
	HasArchive       bool
	CanRebuild       bool
	Processing       bool
	Stage            string
	JobID            uint64
	Error            string
	BrowsePath       string
	ParentURL        string
	Folders          []folder
	BrowseError      string
	MoreFolders      bool
}

// OpenArchive reuses an existing database or presents setup without creating one.
// Missing media is recoverable through the folder selection screen.
func OpenArchive(ctx context.Context, path string) (*Archive, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	a := &Archive{
		ctx: ctx, cancel: cancel, path: path,
		shell: &handler{templates: templates, managed: true},
	}

	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		cancel()

		return nil, fmt.Errorf("open archive: %w", err)
	}

	if err == nil {
		if !info.Mode().IsRegular() {
			cancel()

			return nil, errors.New("archive must be a database file")
		}

		a.store, err = db.Open(path)
		if err != nil {
			cancel()

			return nil, fmt.Errorf("open archive: %w", err)
		}

		a.source, err = a.store.GetExportDirectory(ctx)
		if err != nil {
			a.Close()

			return nil, err
		}

		a.selection = a.source

		imported, err := a.store.HasImport(ctx)
		if err != nil {
			a.Close()

			return nil, err
		}

		var index parser.MediaIndex

		mediaDirectory := ""

		if a.source != "" {
			mediaDirectory = filepath.Join(a.source, "chat_media")

			index, err = parser.NewMediaIndex(mediaDirectory)
		}

		if err != nil {
			a.problem = "The saved export folder is unavailable. Choose its current location or another export to continue."
		} else if imported {
			a.viewer, err = newWithMedia(a.store, mediaDirectory, index, true)
			if err != nil {
				a.Close()

				return nil, err
			}
		}
	}

	assets, err := fs.Sub(files, "assets")
	if err != nil {
		a.Close()

		return nil, err
	}

	mux := http.NewServeMux()

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.HandleFunc("GET /setup", a.setup)
	mux.HandleFunc("GET /setup/status", a.status)
	mux.HandleFunc("POST /archive/import", a.selectFolder)
	mux.HandleFunc("POST /archive/rebuild", a.rebuild)
	mux.HandleFunc("POST /theme", a.shell.setTheme)
	mux.HandleFunc("/", a.serveViewer)

	a.routes = secure(http.NewCrossOriginProtection().Handler(mux))

	return a, nil
}

func (a *Archive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.routes.ServeHTTP(w, r)
}

func (a *Archive) Close() error {
	a.mu.Lock()

	if a.closed {
		a.mu.Unlock()

		return nil
	}

	a.closed = true

	a.cancel()

	a.mu.Unlock()

	a.workers.Wait()

	a.viewMu.Lock()

	defer a.viewMu.Unlock()

	if a.store != nil {
		return a.store.Close()
	}

	return nil
}

func (a *Archive) serveViewer(w http.ResponseWriter, r *http.Request) {
	// Media uses immutable handler state and an already selected source. It does
	// not query the DB, so playback need not block (or wait for) a rebuild.
	if !strings.HasPrefix(r.URL.Path, "/media/") {
		a.viewMu.RLock()

		defer a.viewMu.RUnlock()
	}

	a.mu.RLock()

	viewer, closed := a.viewer, a.closed

	a.mu.RUnlock()

	if closed {
		http.Error(w, "Archive is shutting down.", http.StatusServiceUnavailable)

		return
	}

	if viewer == nil {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)

		return
	}

	viewer.ServeHTTP(w, r)
}

func (a *Archive) snapshot() setupPage {
	a.mu.RLock()

	defer a.mu.RUnlock()

	p := setupPage{
		CurrentDirectory: a.source, Directory: a.selection,
		HasArchive: a.viewer != nil, CanRebuild: a.source != "", Error: a.problem,
	}

	if a.job != nil {
		p.Processing, p.Stage, p.JobID = a.job.running, a.job.stage, a.job.id
	}

	return p
}

func (a *Archive) setup(w http.ResponseWriter, r *http.Request) {
	p := a.snapshot()

	if p.Processing && r.URL.Query().Get("job") != strconv.FormatUint(p.JobID, 10) {
		http.Redirect(w, r, "/setup?job="+strconv.FormatUint(p.JobID, 10), http.StatusSeeOther)

		return
	}

	if job := r.URL.Query().Get("job"); job != "" && job == strconv.FormatUint(p.JobID, 10) && !p.Processing && p.Error == "" && p.HasArchive {
		http.Redirect(w, r, "/", http.StatusSeeOther)

		return
	}

	browse := first(p.Directory, p.CurrentDirectory)

	if browse == "" {
		browse, _ = os.UserHomeDir()
	}

	if browse == "" {
		browse, _ = os.Getwd()
	}

	if requested, ok := r.URL.Query()["browse"]; ok && len(requested) > 0 && !p.Processing {
		browse = first(requested[0], browse)
	}

	if !p.Processing {
		p.browse(browse)
	}

	a.shell.render(w, r, page{Section: "setup", Theme: requestTheme(r), Setup: &p}, http.StatusOK)
}

func folderURL(path string) string {
	return "/setup?" + url.Values{"browse": {path}}.Encode()
}

func (p *setupPage) browse(path string) {
	path, err := filepath.Abs(path)
	if err != nil {
		p.BrowseError = "Unable to resolve this folder. Enter its full path."

		return
	}

	p.BrowsePath = path

	if parent := filepath.Dir(path); parent != path {
		p.ParentURL = folderURL(parent)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		p.BrowseError = "Unable to read this folder. Check the path and permissions."

		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if len(p.Folders) == 200 {
			p.MoreFolders = true

			break
		}

		p.Folders = append(p.Folders, folder{entry.Name(), folderURL(filepath.Join(path, entry.Name()))})
	}
}

func (a *Archive) status(w http.ResponseWriter, r *http.Request) {
	p := a.snapshot()

	w.Header().Set("Cache-Control", "no-store")

	if !p.Processing {
		target := "/setup"

		if p.Error == "" && p.HasArchive {
			target = "/"
		}

		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", target)
		} else {
			http.Redirect(w, r, target, http.StatusSeeOther)
		}

		return
	}

	var body bytes.Buffer

	if err := a.shell.templates.ExecuteTemplate(&body, "setup-status", p); err != nil {
		http.Error(w, "Unable to display import progress.", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, _ = w.Write(body.Bytes())
}

func (a *Archive) selectFolder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to read the selected folder.", http.StatusBadRequest)

		return
	}

	a.beginImport(w, r, r.PostForm.Get("directory"), false)
}

func (a *Archive) rebuild(w http.ResponseWriter, r *http.Request) {
	a.beginImport(w, r, "", true)
}

func (a *Archive) beginImport(w http.ResponseWriter, r *http.Request, directory string, rebuild bool) {
	a.mu.Lock()

	if a.closed {
		a.mu.Unlock()

		http.Error(w, "Archive is shutting down.", http.StatusServiceUnavailable)

		return
	}

	if a.job != nil && a.job.running {
		target := "/setup?job=" + strconv.FormatUint(a.job.id, 10)

		a.mu.Unlock()

		http.Redirect(w, r, target, http.StatusSeeOther)

		return
	}

	if rebuild {
		directory = a.source
	}

	if strings.TrimSpace(directory) == "" {
		a.mu.Unlock()

		http.Error(w, "Select an export folder first.", http.StatusBadRequest)

		return
	}

	id := uint64(1)

	if a.job != nil {
		id = a.job.id + 1
	}

	a.job = &archiveJob{id: id, running: true, stage: "Reading and validating export files", done: make(chan struct{})}

	a.selection, a.problem = directory, ""

	a.workers.Add(1)

	a.mu.Unlock()

	go a.importFolder(directory)

	http.Redirect(w, r, "/setup?job="+strconv.FormatUint(id, 10), http.StatusSeeOther)
}

func (a *Archive) setStage(stage string) {
	a.mu.Lock()

	a.job.stage = stage

	a.mu.Unlock()
}

func (a *Archive) importFolder(directory string) {
	defer a.workers.Done()

	err := a.loadFolder(directory)

	a.mu.Lock()

	defer a.mu.Unlock()

	if err != nil {
		a.problem = err.Error()

		slog.ErrorContext(a.ctx, "import selected export", "error", err)
	}

	a.job.running = false

	close(a.job.done)
}

func (a *Archive) loadFolder(directory string) error {
	if err := a.ctx.Err(); err != nil {
		return err
	}

	backup, err := parser.LoadBackup(directory)
	if err != nil {
		return fmt.Errorf("Could not load this export: %w", err)
	}

	a.setStage("Indexing attachments")

	mediaDirectory := filepath.Join(backup.ExportDirectory, "chat_media")

	index, err := parser.NewMediaIndex(mediaDirectory)
	if err != nil {
		return fmt.Errorf("Could not read this export's attachments: %w", err)
	}

	if err := a.ctx.Err(); err != nil {
		return err
	}

	// Requests see one complete database/media snapshot. Parsing and indexing do
	// not block browsing; requests wait only while the transaction is committed.
	a.viewMu.Lock()

	defer a.viewMu.Unlock()

	store := a.store

	if store == nil {
		store, err = db.Open(a.path)
		if err != nil {
			return fmt.Errorf("Could not create the archive database: %w", err)
		}

		defer func() {
			if a.store != store {
				store.Close()
			}
		}()
	}

	viewer, err := newWithMedia(store, mediaDirectory, index, true)
	if err != nil {
		return fmt.Errorf("Could not prepare the archive viewer: %w", err)
	}

	a.setStage("Saving the archive")

	if err := store.Import(a.ctx, backup); err != nil {
		return fmt.Errorf("Could not save this export; the previous archive is unchanged: %w", err)
	}

	a.mu.Lock()

	a.store, a.viewer, a.source = store, viewer, backup.ExportDirectory

	a.selection = a.source

	a.mu.Unlock()

	return nil
}
