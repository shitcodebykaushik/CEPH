package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed static
var staticFiles embed.FS

var (
	files     map[string][]byte
	fileNames map[string]bool // extensionless paths that are real files
)

func init() {
	files = make(map[string][]byte)
	fileNames = make(map[string]bool)

	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic("web sub: " + err.Error())
	}

	fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(sub, p)
		if err != nil {
			return nil
		}
		files[p] = data
		fileNames[p] = true
		fmt.Printf("[WEB] embedded: %s (%d bytes)\n", p, len(data))
		return nil
	})
}

var Handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/static")
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		p = "index.html"
	}

	data, ok := files[p]
	if !ok {
		p = "index.html"
		data, ok = files[p]
		if !ok {
			http.NotFound(w, r)
			return
		}
	}

	ext := path.Ext(p)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
})
