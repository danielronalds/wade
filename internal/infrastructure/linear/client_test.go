package linear

import "testing"

func TestTicketForBranchBuildsWorkspaceIssueURL(t *testing.T) {
	ticket, err := NewClient().TicketForBranch("Example_Workspace", "feature/abc-123-description")
	if err != nil {
		t.Fatalf("TicketForBranch() error = %v", err)
	}
	if ticket == nil || ticket.Key != "ABC-123" || ticket.URL != "https://linear.app/Example_Workspace/issue/ABC-123" {
		t.Fatalf("TicketForBranch() = %#v", ticket)
	}
}

func TestTicketForBranchHandlesMissingConfigurationAndTicket(t *testing.T) {
	client := NewClient()

	ticket, err := client.TicketForBranch("", "feature/no-ticket")
	if err != nil || ticket != nil {
		t.Fatalf("TicketForBranch() = %#v, %v", ticket, err)
	}
	if _, err := client.TicketForBranch("", "abc-123"); err == nil {
		t.Fatal("TicketForBranch() error = nil")
	}
}
