package flashcard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type FlashcardTemplate struct {
	Name string `yaml:"name"`
	QFmt string `yaml:"qfmt"`
	AFmt string `yaml:"afmt"`
}

type FlashcardField struct {
	Name     string `yaml:"name"`
	Template string `yaml:"template"`
	Markdown bool   `yaml:"markdown"`
}

type FlashcardData struct {
	Filename string   `yaml:"filename"`
	Tags     []string `yaml:"tags"`
}

type FlashcardDeck struct {
	Identifier string `yaml:"identifier"`
	Name       string `yaml:"name"`
}

type FlashcardModel struct {
	Identifier string              `yaml:"identifier"`
	Name       string              `yaml:"name"`
	Type       string              `yaml:"type"`
	Style      string              `yaml:"style"`
	Templates  []FlashcardTemplate `yaml:"templates"`
	Fields     []FlashcardField    `yaml:"fields"`
}

type FlashcardProject struct {
	Identifier string          `yaml:"identifier"`
	Filename   string          `yaml:"filename"`
	Deck       FlashcardDeck   `yaml:"deck"`
	Model      FlashcardModel  `yaml:"model"`
	Data       []FlashcardData `yaml:"data"`
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

	if project.Model.Style, err = filepath.Abs(filepath.Join(directory, project.Model.Style)); err != nil {
		return nil, err
	}
	if _, err := os.Stat(project.Model.Style); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("model style file does not exist: %s", project.Model.Style)
	}

	for _, template := range project.Model.Templates {
		if template.QFmt, err = filepath.Abs(filepath.Join(directory, template.QFmt)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(template.QFmt); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("qfmt file for template %s does not exist: %s", template.Name, template.QFmt)
		}

		if template.AFmt, err = filepath.Abs(filepath.Join(directory, template.AFmt)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(template.AFmt); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("afmt file for template %s does not exist: %s", template.Name, template.AFmt)
		}
	}

	for _, field := range project.Model.Fields {
		if field.Template, err = filepath.Abs(filepath.Join(directory, field.Template)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(field.Template); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("template file for field %s does not exist: %s", field.Name, field.Template)
		}
	}

	for _, data := range project.Data {
		if data.Filename, err = filepath.Abs(filepath.Join(directory, data.Filename)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(data.Filename); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("data file does not exist: %s", data.Filename)
		}
	}

	return project, err
}
