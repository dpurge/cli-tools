package flashcard

import (
	"github.com/dpurge/cli-tools/pkg/tool"
)

func buildAnkiPackage(projectfile string) (string, error) {
	project, err := readProject(projectfile)
	if err != nil {
		return "", err
	}

	pkg := new(tool.AnkiPackage)

	pkgfile, err := pkg.Save(project.Filename)
	if err != nil {
		return "", nil
	}

	return pkgfile, nil
}
