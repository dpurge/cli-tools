package ebook

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type EBookProject struct {
	Identifier string   `yaml:"identifier"`
	Filename   string   `yaml:"filename"`
	Title      string   `yaml:"title"`
	Author     string   `yaml:"author"`
	Language   string   `yaml:"language"`
	Stylesheet []string `yaml:"stylesheet"`
	Text       []string `yaml:"text"`
}

func readProject(filename string) (*EBookProject, error) {

	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	project := &EBookProject{}
	err = yaml.Unmarshal(buf, project)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %w", filename, err)
	}

	filename, err = filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	directory, _ := filepath.Split(filename)
	project.Filename = filepath.Join(directory, project.Filename)

	for i, val := range project.Stylesheet {
		project.Stylesheet[i] = filepath.Join(directory, val)
		_, err = os.Stat(project.Stylesheet[i])
		if errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("stylesheet not found: %s", project.Stylesheet[i])
			return nil, err
		}
	}

	for i, val := range project.Text {
		project.Text[i] = filepath.Join(directory, val)
		_, err = os.Stat(project.Text[i])
		if errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("text not found: %s", project.Text[i])
			return nil, err
		}
	}

	return project, err
}
