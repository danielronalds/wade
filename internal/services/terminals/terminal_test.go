package terminals

// TODO: Review properly

import (
	"reflect"
	"testing"
)

func TestAttachReplaysBufferedOutputBeforeLiveOutput(t *testing.T) {
	terminal := &Terminal{
		buffer:  newOutputBuffer(terminalBufferBytes),
		clients: make(map[*Client]struct{}),
	}
	terminal.buffer.Write([]byte("old output"))

	client := terminal.Attach()
	terminal.broadcast([]byte("new output"))

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
	terminal := &Terminal{
		buffer:  newOutputBuffer(terminalBufferBytes),
		clients: make(map[*Client]struct{}),
	}

	client := terminal.Attach()
	terminal.broadcast([]byte("new output"))

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
