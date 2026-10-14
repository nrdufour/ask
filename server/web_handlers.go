package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"

	"ask/web"
)

type pageData struct {
	Version string
	// AssetHash busts browser caches whenever the embedded assets change.
	AssetHash string
	// Page names the current page, to highlight it in the navigation.
	Page string

	// From the airports import: "83,641", "21 Sep 2025" and the data commit.
	AirportCount string
	DataDate     string
	DataCommit   string
}

var pageNames = []string{"index", "airports", "distance", "reachable"}

// assetHash returns a short hash of every embedded static file.
func assetHash() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(web.FS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(web.FS, path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:12], err
}

// parsePages parses each page together with the shared layout.
func parsePages() (map[string]*template.Template, error) {
	pages := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		t, err := template.ParseFS(web.FS, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, err
		}
		pages[name] = t
	}
	return pages, nil
}

func (s *Server) render(w http.ResponseWriter, name string) {
	data := pageData{Version: VERSION, AssetHash: s.assetHash, Page: name}

	status, err := s.store.ImportStatus()
	if err != nil {
		log.Printf("reading import status: %v", err)
	}
	for _, st := range status {
		if st.TableName != "airports" {
			continue
		}
		data.AirportCount = thousands(st.RecordCount)
		data.DataCommit = st.GitCommitHash
		if t, err := time.Parse(time.RFC3339, st.GitCommitDate); err == nil {
			data.DataDate = t.Format("2 Jan 2006")
		}
	}

	// Render to a buffer so a template error still yields a clean 500.
	var buf bytes.Buffer
	if err := s.pages[name].ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("rendering %s: %v", name, err)
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

func (s *Server) pageHandler(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.render(w, name)
	}
}

// thousands formats n with comma separators: 83641 -> "83,641".
func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
