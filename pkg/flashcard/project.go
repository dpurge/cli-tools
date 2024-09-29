package flashcard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/dpurge/cli-tools/pkg/types"
)

func ReadProject(filename string) (*types.FlashcardProject, error) {

	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	project := &types.FlashcardProject{}
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

	kinds := []string{"normal", "cloze"}
	if !slices.Contains(kinds, project.Model.Kind) {
		return nil, fmt.Errorf("invalid model kind: %s (valid kinds: %v)", project.Model.Kind, kinds)
	}

	if project.Model.Style.CSS, err = filepath.Abs(filepath.Join(directory, project.Model.Style.CSS)); err != nil {
		return nil, err
	}
	if _, err := os.Stat(project.Model.Style.CSS); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("model CSS style file does not exist: %s", project.Model.Style.CSS)
	}

	if project.Model.Style.Latex.Prefix, err = filepath.Abs(filepath.Join(directory, project.Model.Style.Latex.Prefix)); err != nil {
		return nil, err
	}
	if _, err := os.Stat(project.Model.Style.Latex.Prefix); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("model LaTeX prefix file does not exist: %s", project.Model.Style.Latex.Prefix)
	}

	if project.Model.Style.Latex.Postfix, err = filepath.Abs(filepath.Join(directory, project.Model.Style.Latex.Postfix)); err != nil {
		return nil, err
	}
	if _, err := os.Stat(project.Model.Style.Latex.Postfix); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("model LaTeX postfix file does not exist: %s", project.Model.Style.Latex.Postfix)
	}

	for i, template := range project.Model.Templates {
		if template.QFmt, err = filepath.Abs(filepath.Join(directory, template.QFmt)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(template.QFmt); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("qfmt file for template %s does not exist: %s", template.Name, template.QFmt)
		}
		project.Model.Templates[i].QFmt = template.QFmt

		if template.AFmt, err = filepath.Abs(filepath.Join(directory, template.AFmt)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(template.AFmt); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("afmt file for template %s does not exist: %s", template.Name, template.AFmt)
		}
		project.Model.Templates[i].AFmt = template.AFmt
	}

	formats := []string{"text", "markdown"}
	for i, field := range project.Model.Fields {
		if field.Template, err = filepath.Abs(filepath.Join(directory, field.Template)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(field.Template); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("template file for field %s does not exist: %s", field.Name, field.Template)
		}
		project.Model.Fields[i].Template = field.Template

		if !slices.Contains(formats, field.Format) {
			return nil, fmt.Errorf("invalid format in %s field: %s (valid kinds: %v)", field.Name, field.Format, kinds)
		}
	}

	for i, data := range project.Data {
		if data.Filename, err = filepath.Abs(filepath.Join(directory, data.Filename)); err != nil {
			return nil, err
		}
		if _, err := os.Stat(data.Filename); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("data file does not exist: %s", data.Filename)
		}
		project.Data[i].Filename = data.Filename
	}

	return project, err
}
