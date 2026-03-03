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
