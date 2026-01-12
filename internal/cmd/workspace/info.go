package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-chat-api/internal/client"
	"github.com/piekstra/slack-chat-api/internal/output"
)

type infoOptions struct{}

func newInfoCmd() *cobra.Command {
	opts := &infoOptions{}

	return &cobra.Command{
		Use:   "info",
		Short: "Get workspace/team information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(opts, nil)
		},
	}
}

func runInfo(opts *infoOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	team, err := c.GetTeamInfo()
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(team)
	}

	output.KeyValue("ID", team.ID)
	output.KeyValue("Name", team.Name)
	output.KeyValue("Domain", fmt.Sprintf("%s.slack.com", team.Domain))

	return nil
}
