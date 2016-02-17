package tq

import (
	"fmt"
	"github.com/gogits/git-module"
	"path/filepath"
	"strconv"
	"strings"
)

func OpenRepository(dir string) Repository {
	repo, _ := git.OpenRepository(dir)
	return Repository{repo}
}

type Repository struct {
	*git.Repository
}

func (repo Repository) Disambiguate(ref string) string {
	var command = git.NewCommand("rev-parse", fmt.Sprintf("--disambiguate=%s", ref))
	var result, _ = command.RunInDir(repo.Path)
	return strings.TrimSpace(result)
}

func (repo Repository) GetCommitTimestamp(startSha string, endSha string, filename string) int {
	//  git log --pretty=format:%ct -1 $FILE
	var command = git.NewCommand("log", startSha, endSha, "--pretty=format:%ct", "-1", "--", filename)
	var result, _ = command.RunInDir(repo.Path)
	if unixtime, err := strconv.Atoi(strings.TrimSpace(result)); err == nil {
		return unixtime
	}

	return 0
}

func (repo Repository) GetChangedFiles(startSha string, endSha string, filter string, path []string, ext []string) []string {
	var command = git.NewCommand("diff", startSha, endSha, "--name-only", "--no-renames")

	if filter != "" {
		command.AddArguments(fmt.Sprintf("--diff-filter=%s", filter))
	}

	if path != nil {
		command.AddArguments("--")

		for _, value := range path {
			command.AddArguments(value)
		}
	}

	var result, err = command.RunInDir(repo.Path)

	if err != nil {
		fmt.Println(err)
	}

	var validFiles []string

	for _, filename := range strings.Split(result, "\n") {
		if ext != nil {
			fileExtension := filepath.Ext(filename)
			for _, validExtension := range ext {
				if fileExtension == validExtension {
					validFiles = append(validFiles, filename)
				}
			}
		} else {
			validFiles = append(validFiles, filename)
		}
	}

	return validFiles
}
