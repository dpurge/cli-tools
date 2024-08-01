package flashcard

import (
	"github.com/dpurge/cli-tools/pkg/tool"
)

func buildAnkiPackage(projectfile string) (string, error) {
	var err error

	project, err := tool.ReadProject(projectfile)
	if err != nil {
		return "", err
	}

	apkg := tool.NewAnkiPackage()

	err = apkg.Open(project.Filename)
	if err != nil {
		return "", err
	}
	defer apkg.Close()

	err = apkg.LoadProject(project)
	if err != nil {
		return "", err
	}

	return project.Filename, nil
}
