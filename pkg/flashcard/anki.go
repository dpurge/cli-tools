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

	apkg := tool.NewAnkiPackage()

	err = apkg.Open(project.Filename)
	if err != nil {
		return "", err
	}
	defer apkg.Close()

	return project.Filename, nil
}
