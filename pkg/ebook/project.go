package ebook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dpurge/cli-tools/pkg/tool"
	"gopkg.in/yaml.v3"
)

type EBookProject struct {
	Identifier  string      `yaml:"identifier"`
	Filename    string      `yaml:"filename"`
	Title       string      `yaml:"title"`
	Author      string      `yaml:"author,omitempty"`
	Language    string      `yaml:"language,omitempty"`
	Script      string      `yaml:"script,omitempty"`
	Cover       string      `yaml:"cover,omitempty"`
	Description string      `yaml:"description,omitempty"`
	Stylesheet  EBookStyles `yaml:"stylesheet,omitempty"`
	Font        []string    `yaml:"font,omitempty"`
	Image       []string    `yaml:"image,omitempty"`
	Text        [][]string  `yaml:"text,omitempty"`
}

type EBookStyles struct {
	Cover   string   `yaml:"cover,omitempty"`
	Section string   `yaml:"section,omitempty"`
	Chapter string   `yaml:"chapter,omitempty"`
	Common  []string `yaml:"common,omitempty"`
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

	if project.Filename, err = filepath.Abs(filepath.Join(directory, project.Filename)); err != nil {
		return nil, err
	}

	if project.Cover, err = tool.ResolvePath(directory, project.Cover, true); err != nil {
		return nil, err
	}

	if project.Stylesheet.Section, err = tool.ResolvePath(directory, project.Stylesheet.Section, true); err != nil {
		return nil, err
	}

	if project.Stylesheet.Chapter, err = tool.ResolvePath(directory, project.Stylesheet.Chapter, true); err != nil {
		return nil, err
	}

	for i, val := range project.Stylesheet.Common {
		if project.Stylesheet.Common[i], err = tool.ResolvePath(directory, val, true); err != nil {
			return nil, err
		}
	}

	if err = tool.ResolvePaths(directory, project.Font, true); err != nil {
		return nil, err
	}

	if err = tool.ResolvePaths(directory, project.Image, true); err != nil {
		return nil, err
	}

	for _, val := range project.Text {
		if err = tool.ResolvePaths(directory, val, true); err != nil {
			return nil, err
		}
	}

	return project, err
}
