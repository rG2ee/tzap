package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tzapio/tzap/cli/cmd/cmdutil"
	"github.com/tzapio/tzap/pkg/util/stdin"
)

const (
	githubCmdName      = "gh"
	defaultReleaseNote = "# Changelog\n\n"
)

func gitTagExists(tag string) (bool, error) {
	gitCmd := exec.Command("git", "tag", "--list")
	output, err := gitCmd.Output()
	if err != nil {
		return false, err
	}

	tagPattern := regexp.MustCompile("^" + regexp.QuoteMeta(tag) + "\\b")
	tags := strings.Split(string(output), "\n")
	for _, existingTag := range tags {
		if tagPattern.MatchString(existingTag) {
			return true, nil
		}
	}

	return false, nil
}

var ghrelease = &cobra.Command{
	Use:   "ghrelease <tag>",
	Short: "Generate a GitHub release",
	Long: `Generate a GitHub release using ChatGPT.
Prompts ChatGPT to generate release title and release notes based on the diff of the currently staged files.
The release is then created on GitHub using the title and notes generated by ChatGPT.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if exists, err := gitTagExists(args[0]); (err != nil) || !exists {
			cmd.Printf("Tag %s does not exist\n", args[0])
			return
		}

		if exists, err := gitTagExists(args[1]); (err != nil) || !exists {
			cmd.Printf("Tag %s does not exist\n", args[0])
			return
		}
		// Get previous and current tags
		prevTag, currentTag := args[0], args[1]

		// Get git commits from last tag
		commitsCmd := exec.Command("git", "log", "--pretty=format:%s", fmt.Sprintf("%s..HEAD", prevTag))
		commitsOutput, err := commitsCmd.CombinedOutput()
		if err != nil {
			cmd.Println("Could not get git commits:", err)
			return
		}
		cmd.Println("Checking commits:\n", string(commitsOutput))
		// Create title and summary of changes
		commits := strings.Split(strings.TrimSpace(string(commitsOutput)), "\n")
		title := fmt.Sprintf("Release %s", currentTag)
		summary := ""
		for _, commit := range commits {
			summary += fmt.Sprintf("* %s \n", commit)
		}

		url, err := exec.Command("git", "ls-remote", "--get-url").Output()
		if err != nil {
			cmd.Println("Could not get remote URL:", err)
			return
		}

		t := cmdutil.GetTzapFromContext(cmd.Context()).
			AddSystemMessage(fmt.Sprintf(`Be creative and output a GitHub release using the JSON Template. Use titles: Use cases, Features, Changes. Please include the compare tag URL.

Template:
{"title":{title},"notes":{release notes in markdown}}

Repository: ` + string(url))).
			AddUserMessage(fmt.Sprintf("Title: %s\n\nGit Commits:\n%s", title, summary))

		res := t.RequestChatCompletion()

		// Parse the JSON object
		var data map[string]string
		err = json.Unmarshal([]byte(res.Data["content"].(string)), &data)
		if err != nil {
			cmd.Println("Could not parse JSON object:", err)
			return
		}

		// Get title and notes from the JSON object
		notes := data["notes"]
		if !stdin.ConfirmPrompt("Continue with release?") {
			return
		}
		// Create release
		releaseCmd := exec.Command(githubCmdName, "release", "edit", currentTag, "--prerelease", "--title", title, "--notes", notes)
		releaseCmd.Stderr = os.Stderr
		if err := releaseCmd.Run(); err != nil {
			cmd.Printf("Could not create GitHub release. Title: %s, Notes: %s, Error: %s\n", title, notes, err)
			return
		}
	},
}

func init() {
	RootCmd.AddCommand(ghrelease)
}
