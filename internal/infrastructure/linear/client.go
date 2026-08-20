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
type Client struct{}

// NewClient constructs a stateless Linear client.
func NewClient() Client {
	return Client{}
}

// TicketForBranch returns nil when the branch contains no ticket key.
func (Client) TicketForBranch(workspace string, branch string) (*Ticket, error) {
	matches := ticketPattern.FindStringSubmatch(branch)
	if len(matches) < 2 {
		return nil, nil
	}
	if strings.TrimSpace(workspace) == "" {
		return nil, fmt.Errorf("linear workspace is required")
	}

	key := strings.ToUpper(matches[1])
	return &Ticket{
		Key: key,
		URL: fmt.Sprintf("https://linear.app/%s/issue/%s", workspace, key),
	}, nil
}
