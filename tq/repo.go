package tq

import (
	"fmt"
	"github.com/gogits/git-module"
	"path/filepath"
	"strconv"
	"strings"
)

func OpenRepository(dir string) (Repository, error) {
	repo, err := git.OpenRepository(dir)
	if err != nil {
		return Repository{}, fmt.Errorf("Failed to open repository at \"%s\": %s", dir, err)
	}

	return Repository{repo}, err
}

type Repository struct {
	*git.Repository
}

func (repo Repository) Disambiguate(ref string) (string, error) {
	command := git.NewCommand("rev-parse", fmt.Sprintf("--disambiguate=%s", ref))
	result, err := command.RunInDir(repo.Path)
	if err != nil {
		return "", fmt.Errorf("Could not find SHA \"%s\" in repository", ref)
	}
	return strings.TrimSpace(result), nil
}

func (repo Repository) GetCommitTimestamp(startSha string, endSha string, filename string) (int, error) {
	//  git log --pretty=format:%ct -1 $FILE
	var command = git.NewCommand("log", startSha, endSha, "--pretty=format:%ct", "-1", "--", filename)
	result, err := command.RunInDir(repo.Path)
	if err != nil {
		return 0, fmt.Errorf("Could not get modified time for file \"%s\"", filename)
	}

	timeAsString := strings.TrimSpace(result)
	unixtime, err := strconv.Atoi(timeAsString)
	if err != nil {
		return 0, fmt.Errorf("Could not get intermpret timestamp \"%s\"", timeAsString)
	}

	return unixtime, nil
}

func (repo Repository) GetChangedFiles(startSha string, endSha string, filter string, path []string, ext []string) ([]string, error) {
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
		return nil, fmt.Errorf("Failed executing command \"%s\": %s", command, err)
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

	return validFiles, nil
}
