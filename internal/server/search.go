package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/cristianino/mdocs/internal/docs"
)

// SearchIndex es lo que descarga el buscador del visor (/search.json).
type SearchIndex struct {
	Docs []docs.SearchDoc `json:"docs"`
}

// BuildSearchIndex indexa los documentos registrados, en su orden.
func BuildSearchIndex(docList []*docs.Doc) SearchIndex {
	idx := SearchIndex{Docs: []docs.SearchDoc{}}
	for _, d := range docList {
		raw, err := os.ReadFile(d.File)
		if err != nil {
			idx.Docs = append(idx.Docs, docs.SearchDoc{ID: d.ID, Title: d.Title, Subtitle: d.Subtitle, Sections: []docs.SearchSection{}})
			continue
		}
		idx.Docs = append(idx.Docs, docs.IndexDoc(raw, d))
	}
	return idx
}

func handleSearchIndex(w http.ResponseWriter, r *http.Request) {
	_, docList, err := LoadState()
	if err != nil {
		http.Error(w, "Error de configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}
	data, _ := json.Marshal(BuildSearchIndex(docList))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}

// LinkedFiles devuelve los archivos descargables enlazados desde los
// documentos (rutas relativas al proyecto, sin repetir, en orden de
// aparición).
func LinkedFiles(docList []*docs.Doc) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range docList {
		raw, err := os.ReadFile(d.File)
		if err != nil {
			continue
		}
		res, err := docs.Render(raw, docs.Options{Mode: docs.ModeWeb, Doc: d, All: docList})
		if err != nil {
			continue
		}
		for _, f := range res.Files {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	return out
}
