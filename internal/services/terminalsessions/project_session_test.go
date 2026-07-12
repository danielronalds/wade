package terminalsessions

// TODO: Review properly

import (
	"reflect"
	"testing"
)

func TestAttachReplaysBufferedOutputBeforeLiveOutput(t *testing.T) {
	projectSession := &ProjectSession{
		buffer:  newOutputBuffer(projectSessionBufferBytes),
		clients: make(map[*Client]struct{}),
	}
	projectSession.buffer.Write([]byte("old output"))

	client := projectSession.Attach()
	projectSession.broadcast([]byte("new output"))

	outputs := readClientOutputs(t, client, 4)
	want := []ClientOutput{
		{Kind: ClientOutputKindReplayStart},
		{Kind: ClientOutputKindData, Data: []byte("old output")},
		{Kind: ClientOutputKindReplayEnd},
		{Kind: ClientOutputKindData, Data: []byte("new output")},
	}

	assertClientOutputs(t, outputs, want)
}

func TestAttachWithoutBufferedOutputDoesNotSendReplayMarkers(t *testing.T) {
	projectSession := &ProjectSession{
		buffer:  newOutputBuffer(projectSessionBufferBytes),
		clients: make(map[*Client]struct{}),
	}

	client := projectSession.Attach()
	projectSession.broadcast([]byte("new output"))

	outputs := readClientOutputs(t, client, 1)
	want := []ClientOutput{{Kind: ClientOutputKindData, Data: []byte("new output")}}

	assertClientOutputs(t, outputs, want)
}

func readClientOutputs(t *testing.T, client *Client, count int) []ClientOutput {
	t.Helper()

	outputs := make([]ClientOutput, 0, count)
	for range count {
		select {
		case output := <-client.Output():
			outputs = append(outputs, output)
		default:
			t.Fatalf("client output length = %d, want %d", len(outputs), count)
		}
	}

	select {
	case output := <-client.Output():
		t.Fatalf("unexpected client output: %#v", output)
	default:
	}

	return outputs
}

func assertClientOutputs(t *testing.T, got []ClientOutput, want []ClientOutput) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("outputs = %#v, want %#v", got, want)
	}
}

func TestShouldStartAgentCommandOnlyForAgentTerminal(t *testing.T) {
	tests := map[string]struct {
		terminalName string
		agentCommand string
		want         bool
	}{
		"agent terminal with command": {
			terminalName: agentTerminalName,
			agentCommand: "pi -c",
			want:         true,
		},
		"agent terminal without command": {
			terminalName: agentTerminalName,
			agentCommand: "",
			want:         false,
		},
		"misc terminal with command": {
			terminalName: "misc",
			agentCommand: "pi -c",
			want:         false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := shouldStartAgentCommand(test.terminalName, test.agentCommand)
			if got != test.want {
				t.Fatalf("shouldStartAgentCommand() = %v, want %v", got, test.want)
			}
		})
	}
}
