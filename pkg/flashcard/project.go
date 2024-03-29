package flashcard

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type FlashcardProject struct {
	Identifier string `yaml:"identifier"`
	Filename   string `yaml:"filename"`
}

func readProject(filename string) (*FlashcardProject, error) {

	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	project := &FlashcardProject{}
	err = yaml.Unmarshal(buf, project)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %w", filename, err)
	}

	filename, err = filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	directory, _ := filepath.Split(filename)

	if project.Filename, err = filepath.Abs(filepath.Join(directory, project.Filename)); err != nil {
		return nil, err
	}

	return project, err
}
