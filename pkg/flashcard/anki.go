package flashcard

import (
	"github.com/dpurge/cli-tools/pkg/tool"
)

func buildAnkiPackage(projectfile string) (string, error) {
	var err error

	project, err := readProject(projectfile)
	if err != nil {
		return "", err
	}

	apkg, err := tool.NewAnkiPackage()
	if err != nil {
		return "", err
	}
	defer apkg.Save(project.Filename)

	return project.Filename, err
}
