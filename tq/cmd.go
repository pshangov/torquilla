package tq

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/gogs/git-module"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"sort"
	"strings"
    "text/template"
)

type TemplateData struct {
	Script string
	Sha string
}

var RootCmd = &cobra.Command{
	Use:   "tq start_commit [end_commit]",
	Short: "Torquilla is a ligtweight schema migration tool",
	Long:  "Torquilla is a ligtweight schema migration tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := run(args)
		return err
	},
}

func init() {
	viper.SetConfigName("torquilla")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Sprintf("Fatal error config file: %s \n", err)
	}

	var workingDir, _ = os.Getwd()
	RootCmd.PersistentFlags().StringP("dir", "d", workingDir, "repository working directory")
	viper.BindPFlag("dir", RootCmd.PersistentFlags().Lookup("dir"))

	RootCmd.PersistentFlags().StringP("manifest", "m", "", "manifest file")
	viper.BindPFlag("manifest", RootCmd.PersistentFlags().Lookup("manifest"))

	RootCmd.PersistentFlags().BoolP("silent", "s", false, "don't display warnings")
	viper.BindPFlag("silent", RootCmd.PersistentFlags().Lookup("silent"))

	RootCmd.PersistentFlags().BoolP("version", "v", false, "print version number and exit")
	viper.BindPFlag("version", RootCmd.PersistentFlags().Lookup("version"))

	RootCmd.PersistentFlags().BoolP("name-only", "n", false, "only print filenames")
	viper.BindPFlag("name-only", RootCmd.PersistentFlags().Lookup("name-only"))
}

func run(args []string) error {
	if viper.GetBool("version") {
		fmt.Println("0.02")
		return nil
	}

	var startSha, endSha string
	repo, err := OpenRepository(viper.GetString("dir"))
	if err != nil {
		return err
	}

	switch len(args) {
	case 0:
		return errors.New("Please provide commit range to work on")
	case 1:
		startSha, err = repo.Disambiguate(args[0])
		if err != nil {
			return err
		}

		endSha, err = repo.RevParse("HEAD")
		if err != nil {
			return err
		}
	case 2:
		startSha, err = repo.Disambiguate(args[0])
		if err != nil {
			return err
		}
		endSha, err = repo.Disambiguate(args[1])
		if err != nil {
			return err
		}
	default:
		return errors.New("Too many arguments provided")
	}

	endCommitTree, err := repo.LsTree(endSha)
	if err != nil {
		return fmt.Errorf("Failed fetching SHA %s from git", endSha)
	}

	changedFilenames, err := getChangedFilenames(repo, startSha, endSha) // FIXME
	if err != nil {
		return err
	}

	scripts, err := loadScripts(repo, *endCommitTree, startSha, endSha, changedFilenames) // FIXME
	if err != nil {
		return err
	}

	if viper.GetBool("name-only") {
		for _, script := range scripts {
			fmt.Println(script.Name)
		}
		return nil
	}

	output := concatenateScripts(scripts, endSha)

	if viper.GetString("template") != "" {
        tmpl, err := template.New("migration").Parse(viper.GetString("template"))
        if err != nil {
            return err
        }

        err = tmpl.Execute(os.Stdout, TemplateData{output, endSha})
        if err != nil {
            return err
        }
	} else {
        fmt.Println(output)
    }


	return nil
}

func getChangedFilenames(repo Repository, startSha string, endSha string) ([]string, error) {
	var changedFilenames []string

	if len(viper.GetStringSlice("migrations")) != 0 {
		viper.GetStringSlice("migrations")
		changedMigrations, err := repo.GetChangedFiles(startSha, endSha, "A", viper.GetStringSlice("migrations"), viper.GetStringSlice("extensions"))
		if err != nil {
			return changedFilenames, err
		}
		changedFilenames = append(changedFilenames, changedMigrations...)
	}

	if len(viper.GetStringSlice("definitions")) != 0 {
		viper.GetStringSlice("definitions")
		changedDefinitions, err := repo.GetChangedFiles(startSha, endSha, "AM", viper.GetStringSlice("definitions"), viper.GetStringSlice("extensions"))
		if err != nil {
			return changedFilenames, err
		}
		changedFilenames = append(changedFilenames, changedDefinitions...)
	}

	return changedFilenames, nil
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func loadScripts(repo Repository, tree git.Tree, startSha string, endSha string, filenames []string) ([]Script, error) {
	var scripts []Script
	var manifestLines []string

	manifest := viper.GetString("manifest")

	if manifest != "" {
		lines, err := readLines(manifest)
		if err != nil {
			return nil, fmt.Errorf("Could not read manifest file \"%s\"", manifest)
		}
		manifestLines = append(manifestLines, lines...)
	}

	for _, filename := range filenames {
		blob, err := tree.Blob(filename)
		if err != nil {
			return nil, err
		}

		data, err := blob.Bytes()
		if err != nil {
			return nil, err
		}

		timestamp, err := repo.GetCommitTimestamp(startSha, endSha, filename)
		if err != nil {
			return nil, err
		}

		if manifest != "" {
			found := false
			for pos, val := range manifestLines {
				if val == filename {
					scripts = append(scripts, Script{filename, timestamp, pos, string(data)})
					found = true
				}
			}
			if !found && !viper.GetBool("silent") {
				fmt.Fprintf(os.Stderr, "Changed file not found in manifest: %s\n", filename)
			}
		} else {
			scripts = append(scripts, Script{filename, timestamp, 0, string(data)})
		}

	}

	if manifest != "" {
		sort.Sort(ByManifest(scripts))
	} else {
		sort.Sort(ByAge(scripts))
	}

	return scripts, nil
}

func concatenateScripts(scripts []Script, sha string) string {
	var entries []string

	for _, script := range scripts {
		header := fmt.Sprintf("-- %s:%s", sha, script.Name)
		entries = append(entries, header, script.Data)
	}

	output := strings.Join(entries, "\n\n")

	return output
}
