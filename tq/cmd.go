package tq

import (
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

// 1. DONE: Implement sorting
// 2. WORK: Read configuration script
// 3. WORK: Add support for filtering by extension
// 4. Add support for filename output only
// 5. Proper error handling
// 6. Git-compliant revision handling
// 7. Refactor code

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
}

func run(args []string) error {
	var startSha, endSha string
	repo := OpenRepository(viper.GetString("dir"))

	switch len(args) {
	case 0:
		return errors.New("Please provide commit range to work on")
	case 1:
		startSha = repo.Disambiguate(args[0])
		var head, _ = repo.GetHEADBranch()
		endSha, _ = repo.GetBranchCommitID(head.Name)
	case 2:
		startSha = repo.Disambiguate(args[0])
		endSha = repo.Disambiguate(args[1])
	default:
		return errors.New("Too many arguments provided")
	}

	var endCommit, _ = repo.GetCommit(endSha)

	changedFilenames := getChangedFilenames(repo, startSha, endSha)
	scripts := loadScripts(repo, endCommit.Tree, startSha, endSha, changedFilenames)
	output := concatenateScripts(scripts, endSha)

	fmt.Println(output)

	return nil
}

func getChangedFilenames(repo Repository, startSha string, endSha string) []string {
	var changedFilenames []string

	changedMigrations := repo.GetChangedFiles(startSha, endSha, "A", viper.GetStringSlice("migrations"), viper.GetStringSlice("extensions"))
	changedDefinitions := repo.GetChangedFiles(startSha, endSha, "AM", viper.GetStringSlice("definitions"), viper.GetStringSlice("extensions"))

	changedFilenames = append(changedFilenames, changedMigrations...)
	changedFilenames = append(changedFilenames, changedDefinitions...)

	return changedFilenames
}

func loadScripts(repo Repository, tree git.Tree, startSha string, endSha string, filenames []string) []Script {
	var scripts []Script

	for _, filename := range filenames {
		blob, _ := tree.GetBlobByPath(filename)
		data, _ := blob.Data()
		buffer := new(bytes.Buffer)
		buffer.ReadFrom(data)
		timestamp := repo.GetCommitTimestamp(startSha, endSha, filename)

		scripts = append(scripts, Script{filename, timestamp, buffer.String()})
	}

	sort.Sort(ByAge(scripts))

	return scripts
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
