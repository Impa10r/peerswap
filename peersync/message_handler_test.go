package peersync

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/elementsproject/peerswap/messages"
)

func TestCapabilityHandlerIgnoresSwapMessages(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	h := &messageHandler{}
	for _, kind := range []messages.MessageType{
		messages.MESSAGETYPE_SWAPINREQUEST, messages.MESSAGETYPE_SWAPOUTREQUEST,
		messages.MESSAGETYPE_SWAPINAGREEMENT, messages.MESSAGETYPE_SWAPOUTAGREEMENT,
		messages.MESSAGETYPE_OPENINGTXBROADCASTED, messages.MESSAGETYPE_CANCELED,
		messages.MESSAGETYPE_COOPCLOSE,
	} {
		h.processMessage(context.Background(), CustomMessage{Type: kind})
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected log: %s", output.String())
	}
	h.processMessage(context.Background(), CustomMessage{Type: 1})
	if !bytes.Contains(output.Bytes(), []byte("unknown message type")) {
		t.Fatal("unknown type should still be logged")
	}
}
