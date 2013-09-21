// Package combo implements an http.Handler that serves concatenated files in a single request.
package combo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Handler struct {
	// Root file system.
	Root http.FileSystem

	// URL path (unrestricted if empty).
	URLPath string

	// Separator of file names in request query (e.g. "&").
	Separator string

	// Maximum number of files to concatenate.
	// If more requested, returns an Bad Request error.
	MaxFiles int
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Bad request method", http.StatusBadRequest)
		return
	}
	if h.URLPath != "" && h.URLPath != r.URL.Path {
		http.NotFound(w, r)
		return
	}
	filenames := strings.Split(r.URL.RawQuery, h.Separator)
	if len(filenames) == 0 {
		http.NotFound(w, r)
		return
	}
	if len(filenames) > h.MaxFiles {
		http.Error(w, fmt.Sprintf("Too many files: %d", len(filenames)), http.StatusBadRequest)
		return
	}
	// Unescape names.
	for i, escapedName := range filenames {
		name, err := url.QueryUnescape(escapedName)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		filenames[i] = name
	}
	if !stringsUnique(filenames) {
		http.Error(w, "Request contains repeated name(s)", http.StatusBadRequest)
		return
	}
	h.serveFiles(w, r, filenames)
}

func stringsUnique(strs []string) bool {
	set := make(map[string]struct{}, len(strs))
	for _, s := range strs {
		if _, ok := set[s]; ok {
			return false
		}
		set[s] = struct{}{}
	}
	return true
}

func (h *Handler) serveFiles(w http.ResponseWriter, r *http.Request, filenames []string) {
	var buf bytes.Buffer
	var maxModTime time.Time
	for _, name := range filenames {
		err := h.appendFileContent(&buf, &maxModTime, name)
		if err != nil && os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("File not found: %q", name), http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	http.ServeContent(w, r, filenames[0], maxModTime, bytes.NewReader(buf.Bytes()))
}

func (h *Handler) appendFileContent(buf *bytes.Buffer, maxModTime *time.Time, filename string) error {
	// Open file.
	f, err := h.Root.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Check that file is not a directory.
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if fi.IsDir() {
		// We do not serve directories, so it's "not found" error.
		return os.ErrNotExist
	}

	// Update max modification time.
	t := fi.ModTime()
	if t.After(*maxModTime) {
		*maxModTime = t
	}

	// Add file contents to the buffer.
	if _, err := io.Copy(buf, f); err != nil {
		return err
	}
	return nil
}
