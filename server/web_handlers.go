package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ask/store"
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

	// Tiles is set when the self-hosted basemap is ready to serve.
	Tiles bool

	// For the airport page: the airport, or the code nothing matched.
	Detail   *store.AirportDetail
	NotFound string
}

var pageNames = []string{"index", "airports", "distance", "reachable", "airport"}

var templateFuncs = template.FuncMap{
	"thousands": thousands,
	"feetToMeters": func(ft int) string {
		return thousands(int(math.Round(float64(ft) * 0.3048)))
	},
	// heading formats a course as three digits: 85.4 -> "085°".
	"heading": func(deg float64) string {
		return fmt.Sprintf("%03.0f°", math.Mod(math.Round(deg)+360, 360))
	},
	"typeLabel": func(t string) string {
		return strings.ToUpper(t[:1]) + strings.ReplaceAll(t[1:], "_", " ")
	},
	// navaidFrequency shows VHF navaids in MHz and NDBs in kHz; DME and
	// TACAN are tuned by channel.
	"navaidFrequency": func(n store.Navaid) string {
		switch {
		case n.Type == "DME" || n.Type == "TACAN":
			return n.DMEChannel
		case n.FrequencyKHz == nil:
			return ""
		case strings.HasPrefix(n.Type, "NDB"):
			return fmt.Sprintf("%d kHz", *n.FrequencyKHz)
		default:
			return strconv.FormatFloat(float64(*n.FrequencyKHz)/1000, 'f', 2, 64) + " MHz"
		}
	},
	"mhz":        func(f float64) string { return strconv.FormatFloat(f, 'f', 3, 64) },
	"deref":      func(p *int) int { return *p },
	"derefFloat": func(p *float64) float64 { return *p },
}

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
		t, err := template.New("").Funcs(templateFuncs).ParseFS(web.FS, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, err
		}
		pages[name] = t
	}
	return pages, nil
}

func (s *Server) render(w http.ResponseWriter, name string, status int, data pageData) {
	data.Version, data.AssetHash, data.Page = VERSION, s.assetHash, name
	data.Tiles = s.tiles.status() == tilesReady

	imports, err := s.store.ImportStatus()
	if err != nil {
		log.Printf("reading import status: %v", err)
	}
	for _, st := range imports {
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
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (s *Server) pageHandler(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.render(w, name, http.StatusOK, pageData{})
	}
}

func (s *Server) airportPageHandler(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	detail, err := s.store.Detail(code)
	var inputErr *store.InputError
	switch {
	case errors.Is(err, store.ErrNotFound) || errors.As(err, &inputErr):
		s.render(w, "airport", http.StatusNotFound, pageData{NotFound: code})
	case err != nil:
		log.Printf("airport page %s: %v", code, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	default:
		s.render(w, "airport", http.StatusOK, pageData{Detail: detail})
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
