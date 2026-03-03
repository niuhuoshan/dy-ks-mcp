package platform

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SelectorSet struct {
	Platform  string            `yaml:"platform"`
	Selectors map[string]string `yaml:"selectors"`
}

func LoadSelectorFile(path string) (SelectorSet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return SelectorSet{}, fmt.Errorf("read selector file: %w", err)
	}
	var s SelectorSet
	if err := yaml.Unmarshal(b, &s); err != nil {
		return SelectorSet{}, fmt.Errorf("parse selector yaml: %w", err)
	}
	if s.Selectors == nil {
		s.Selectors = map[string]string{}
	}
	return s, nil
}
