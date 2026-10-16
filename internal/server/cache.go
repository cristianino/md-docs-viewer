package server

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Las exportaciones con Chrome son costosas: se generan de a una (evita que
// varias peticiones simultáneas lancen varios Chrome en un VPS pequeño) y se
// cachean en memoria hasta que cambie algún archivo del proyecto.
var (
	exportMu    sync.Mutex
	exportCache = map[string]cacheEntry{}
)

type cacheEntry struct {
	fingerprint string
	data        []byte
}

// projectFingerprint resume nombre, tamaño y fecha de los archivos que
// afectan las exportaciones.
func projectFingerprint() string {
	h := sha256.New()
	add := func(p string) {
		if st, err := os.Stat(p); err == nil {
			fmt.Fprintf(h, "%s|%d|%d\n", p, st.Size(), st.ModTime().UnixNano())
		}
	}
	add("brand.config.json")
	add("docs.config.json")
	add("logo.svg")
	if _, docList, err := LoadState(); err == nil {
		for _, d := range docList {
			add(d.File)
		}
	}
	for _, dir := range []string{"templates", "docs"} {
		filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
			if err == nil && !e.IsDir() {
				add(p)
			}
			return nil
		})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func cachedExport(key string, gen func() ([]byte, error)) ([]byte, error) {
	exportMu.Lock()
	defer exportMu.Unlock()
	fp := projectFingerprint()
	if e, ok := exportCache[key]; ok && e.fingerprint == fp {
		return e.data, nil
	}
	data, err := gen()
	if err != nil {
		return nil, err
	}
	exportCache[key] = cacheEntry{fingerprint: fp, data: data}
	return data, nil
}
