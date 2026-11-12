package websocket

import (
	"bufio"
	"bytes"
	"net"
	"testing"
	"time"
)

func serverFrame(op byte, fin bool, payload []byte) []byte {
	head := op
	if fin {
		head |= finBit
	}

	return append([]byte{head, byte(len(payload))}, payload...)
}

func TestReadJoinsFragmentsAndAnswersPing(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	c := &Conn{conn: client, r: bufio.NewReader(client)}

	go func() {
		server.Write(serverFrame(OpText, false, []byte(`{"a":`)))
		server.Write(serverFrame(opPing, true, []byte("hi")))

		pong := make([]byte, 64)
		n, _ := server.Read(pong)

		srv := &Conn{conn: server, r: bufio.NewReader(bytes.NewReader(pong[:n]))}
		_, op, payload, _ := srv.readFrame()

		if op != opPong || string(payload) != "hi" {
			server.Write(serverFrame(opClose, true, nil))

			return
		}

		server.Write(serverFrame(opContinue, true, []byte(`1}`)))
	}()

	got, err := c.Read(time.Second)
	if err != nil || string(got) != `{"a":1}` {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestFrameMasksClientPayload(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 300)
	f := frame(OpText, payload)

	c := &Conn{r: bufio.NewReader(bytes.NewReader(f))}

	fin, op, got, err := c.readFrame()
	if err != nil || !fin || op != OpText || !bytes.Equal(got, payload) || f[1]&maskBit == 0 {
		t.Fatalf("fin %v op %v len %d err %v", fin, op, len(got), err)
	}
}

func TestAcceptKeyFromRFC(t *testing.T) {
	if got := accept("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("got %q", got)
	}
}
