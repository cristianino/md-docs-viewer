package docs

import (
	"encoding/json"
	"fmt"
	"os"
)

type Doc struct {
	ID       string `json:"id"`
	File     string `json:"file"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

func Load(path string) ([]*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("docs config: %w", err)
	}
	var docs []*Doc
	if err := json.Unmarshal(data, &docs); err != nil {
		return nil, fmt.Errorf("docs config: %w", err)
	}
	return docs, nil
}

// Save escribe la lista de documentos en formato JSON indentado.
func Save(path string, list []*Doc) error {
	if list == nil {
		list = []*Doc{}
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
