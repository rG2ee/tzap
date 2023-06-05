package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/tzapio/tzap/cli/cmd/cmdutil"
	"github.com/tzapio/tzap/pkg/types"
	"github.com/tzapio/tzap/pkg/tzap"
	"github.com/tzapio/tzap/workflows/code/git"
	"github.com/tzapio/tzap/workflows/code/gocode"
	"github.com/tzapio/tzap/workflows/stdinworkflows"
	"github.com/tzapio/tzap/workflows/truncate"
)

var showDiff bool
var semanticGitcommitCmd = &cobra.Command{
	Aliases: []string{"c", "commit"},
	Use:     "semantic:gitcommit [clarifying prompt]",
	Short:   "Generate a git commit message using ChatGPT",
	Long:    `Prompts ChatGPT to generate a commit message and commits it to the current git repo. The generated commit message is based on the diff of the currently staged files.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := tzap.HandlePanic(func() {
			t := cmdutil.GetTzapFromContext(cmd.Context())
			defer t.HandleShutdown()
			t.
				ApplyWorkflow(gocode.DeserializedArguments("extraPrompt", args)).
				ApplyErrorWorkflow(git.GitDiff(), func(et *tzap.ErrorTzap) error {
					return et.Err
				}).
				WorkTzap(func(t *tzap.Tzap) {
					diff := t.Data["git-diff"].(string)
					cmd.Print("Reading staged git commit diffs")
					if !showDiff {
						cmd.Println(" (Use --show-diff to show the git diff)\n\n")
					} else {
						cmd.Println(":")
					}
					time.Sleep(500 * time.Millisecond)
					if showDiff {
						cmd.Println()
						cmd.Println(diff)
						cmd.Println("\n\n")
					}
				}).
				ApplyErrorWorkflow(git.ValidateDiff(), func(et *tzap.ErrorTzap) error {
					return et.Err
				}).
				ApplyWorkflow(truncate.SetContextSize()).
				ApplyErrorWorkflow(truncate.CountTokens(), func(et *tzap.ErrorTzap) error {
					return et.Err
				}).
				ApplyErrorWorkflow(truncate.TruncateTokens(), func(et *tzap.ErrorTzap) error {
					return et.Err
				}).
				ApplyErrorWorkflow(RequestChat(), func(et *tzap.ErrorTzap) error {
					return et.Err
				}).
				ApplyErrorWorkflow(gocode.DisplayAndConfirm(), func(et *tzap.ErrorTzap) error {
					return et.Err
				}).
				ApplyErrorWorkflow(git.GitCommit(), func(et *tzap.ErrorTzap) error {
					return et.Err
				})
		})
		if err != nil {
			println(err.Error())
		}
	},
}

// RequestChat is a workflow that requests a chat from ChatGPT.
func RequestChat() types.NamedWorkflow[*tzap.Tzap, *tzap.ErrorTzap] {
	return types.NamedWorkflow[*tzap.Tzap, *tzap.ErrorTzap]{
		Name: "RequestChat",
		Workflow: func(t *tzap.Tzap) *tzap.ErrorTzap {
			extraPrompt := t.Data["extraPrompt"].(string)
			diff := t.Data["git-diff"].(string)
			t = t.AddSystemMessage(`Write one commit using semantic commit specification. \n\n` + CV100)
			if extraPrompt != "" {
				t = t.AddUserMessage(extraPrompt)
			}
			return t.AddUserMessage(diff).
				RequestChatCompletion().
				ApplyWorkflow(stdinworkflows.BeforeProceedingWorkflow()).
				ErrorTzap(nil)
		}}
}

func init() {
	RootCmd.AddCommand(semanticGitcommitCmd)
	// add flag to show git diff
	semanticGitcommitCmd.Flags().BoolVarP(&showDiff, "show-diff", "d", false, "Show git diff")
}
