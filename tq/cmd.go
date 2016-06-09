package tq

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/gogits/git-module"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"sort"
	"strings"
)

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
		fmt.Println("0.01")
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

		head, err := repo.GetHEADBranch()
		if err != nil {
			return err
		}

		endSha, err = repo.GetBranchCommitID(head.Name)
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

	endCommit, err := repo.GetCommit(endSha)
	if err != nil {
		return fmt.Errorf("Failed fetching SHA %s from git", endSha)
	}

	changedFilenames, err := getChangedFilenames(repo, startSha, endSha) // FIXME
	if err != nil {
		return err
	}

	scripts, err := loadScripts(repo, endCommit.Tree, startSha, endSha, changedFilenames) // FIXME
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

	fmt.Println(output)

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
		blob, err := tree.GetBlobByPath(filename)
		if err != nil {
			return nil, err
		}

		data, err := blob.Data()
		if err != nil {
			return nil, err
		}

		buffer := new(bytes.Buffer)
		buffer.ReadFrom(data)
		timestamp, err := repo.GetCommitTimestamp(startSha, endSha, filename)
		if err != nil {
			return nil, err
		}

		if manifest != "" {
			found := false
			for pos, val := range manifestLines {
				if val == filename {
					scripts = append(scripts, Script{filename, timestamp, pos, buffer.String()})
					found = true
				}
			}
			if !found && !viper.GetBool("silent") {
				fmt.Fprintf(os.Stderr, "Changed file not found in manifest: %s\n", filename)
			}
		} else {
			scripts = append(scripts, Script{filename, timestamp, 0, buffer.String()})
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

	if viper.GetString("version_tmpl") != "" {
		entries = append(
			entries,
			"-- update version number",
			fmt.Sprintf(viper.GetString("version_tmpl"), sha),
		)
	}

	output := strings.Join(entries, "\n\n")

	return output
}
