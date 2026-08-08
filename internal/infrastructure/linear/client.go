package linear

import (
	"fmt"
	"regexp"
	"strings"
)

var ticketPattern = regexp.MustCompile(`([a-zA-Z]+-[0-9]+)`)

// Ticket is provider data for an issue associated with a branch.
type Ticket struct {
	Key string
	URL string
}

// Client resolves Linear ticket references encoded in branch names.
type Client struct {
	workspace string
}

// NewClient constructs a Linear client for a workspace slug.
func NewClient(workspace string) Client {
	return Client{workspace: workspace}
}

// TicketForBranch returns nil when the branch contains no ticket key.
func (client Client) TicketForBranch(branch string) (*Ticket, error) {
	matches := ticketPattern.FindStringSubmatch(branch)
	if len(matches) < 2 {
		return nil, nil
	}
	if strings.TrimSpace(client.workspace) == "" {
		return nil, fmt.Errorf("Linear workspace is required")
	}

	key := strings.ToUpper(matches[1])
	return &Ticket{
		Key: key,
		URL: fmt.Sprintf("https://linear.app/%s/issue/%s", client.workspace, key),
	}, nil
}
