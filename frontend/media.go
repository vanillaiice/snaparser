package frontend

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vanillaiice/snaparser/v2/parser"
)

type attachment struct {
	URL  string
	Kind string
}

func mediaKind(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif":
		return "image"

	case ".mp4", ".webm", ".mov":
		return "video"

	case ".mp3", ".ogg", ".wav", ".m4a", ".aac", ".flac":
		return "audio"

	default:
		return "download"
	}
}

func (h *handler) attachments(chat parser.Chat) []attachment {
	var items []attachment

	for _, path := range chat.ResolveMediaPaths(h.mediaIndex) {
		name := filepath.Base(path)

		kind := mediaKind(name)

		// Snapchat stores voice notes in MP4 containers too. The message's
		// NOTE type distinguishes those audio-only files from actual videos.
		if kind == "video" && strings.EqualFold(strings.TrimSpace(chat.MediaType), "NOTE") {
			kind = "audio"
		}

		target := "/media/" + url.PathEscape(name)

		if h.mediaVersion != "" {
			target += "?v=" + h.mediaVersion
		}

		items = append(items, attachment{URL: target, Kind: kind})
	}

	return items
}

func (h *handler) serveMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")

	name := r.PathValue("filename")

	// An older page must not load a different export's file with the same name.
	if version := r.URL.Query().Get("v"); version != "" && version != h.mediaVersion {
		http.NotFound(w, r)

		return
	}

	if h.mediaDirectory == "" || name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		http.NotFound(w, r)

		return
	}

	// Constrain access even if a file has been replaced with a symlink since
	// startup. Files outside chat_media and directory listings are never served.
	file, err := os.OpenInRoot(h.mediaDirectory, name)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Unable to read attachment.", http.StatusInternalServerError)
		}

		return
	}

	defer file.Close()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)

		return
	}

	if mediaKind(name) == "download" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	} else if contentType := mime.TypeByExtension(filepath.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeContent(w, r, name, info.ModTime(), file)
}
