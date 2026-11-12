package websocket

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	acceptGUID    = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	maxFrame      = 1 << 20
	opContinue    = 0x0
	OpText        = 0x1
	opClose       = 0x8
	opPing        = 0x9
	opPong        = 0xA
	finBit        = 0x80
	maskBit       = 0x80
	shortLength   = 125
	mediumLength  = 126
	longLength    = 127
	writeDeadline = 10 * time.Second
)

var ErrClosed = errors.New("websocket closed by server")

type Conn struct {
	conn net.Conn
	r    *bufio.Reader
	mu   sync.Mutex
}

func Dial(ctx context.Context, rawURL string, header http.Header) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	dialer := tls.Dialer{Config: &tls.Config{ServerName: u.Hostname(), MinVersion: tls.VersionTLS12}}

	nc, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(u.Hostname(), "443"))
	if err != nil {
		return nil, err
	}

	c := &Conn{conn: nc, r: bufio.NewReader(nc)}

	if err := c.handshake(u, header); err != nil {
		nc.Close()

		return nil, err
	}

	return c, nil
}

func (c *Conn) handshake(u *url.URL, header http.Header) error {
	key := make([]byte, 16)
	rand.Read(key)

	encoded := base64.StdEncoding.EncodeToString(key)

	req := &http.Request{
		Method:     http.MethodGet,
		URL:        &url.URL{Scheme: "https", Host: u.Host, Path: u.Path, RawQuery: u.RawQuery},
		Host:       u.Host,
		Header:     header.Clone(),
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", encoded)
	req.Header.Set("Sec-WebSocket-Version", "13")

	if err := req.Write(c.conn); err != nil {
		return err
	}

	resp, err := http.ReadResponse(c.r, req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()

		return fmt.Errorf("websocket handshake: %s %s", resp.Status, body)
	}

	if resp.Header.Get("Sec-WebSocket-Accept") != accept(encoded) {
		return errors.New("websocket handshake: bad accept key")
	}

	return nil
}

func accept(key string) string {
	sum := sha1.Sum([]byte(key + acceptGUID))

	return base64.StdEncoding.EncodeToString(sum[:])
}

func (c *Conn) Read(timeout time.Duration) ([]byte, error) {
	var message []byte

	for {
		if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return nil, err
		}

		fin, op, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}

		switch op {
		case opPing:
			if err := c.write(opPong, payload); err != nil {
				return nil, err
			}
		case opPong:
		case opClose:
			return nil, ErrClosed
		case OpText, opContinue:
			message = append(message, payload...)

			if fin {
				return message, nil
			}
		}
	}
}

func (c *Conn) readFrame() (bool, byte, []byte, error) {
	var head [2]byte
	if _, err := io.ReadFull(c.r, head[:]); err != nil {
		return false, 0, nil, err
	}

	length, err := c.readLength(head[1] &^ maskBit)
	if err != nil {
		return false, 0, nil, err
	}

	if length > maxFrame {
		return false, 0, nil, fmt.Errorf("websocket frame of %d bytes", length)
	}

	var mask [4]byte
	if head[1]&maskBit != 0 {
		if _, err := io.ReadFull(c.r, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.r, payload); err != nil {
		return false, 0, nil, err
	}

	if head[1]&maskBit != 0 {
		applyMask(payload, mask)
	}

	return head[0]&finBit != 0, head[0] & 0x0F, payload, nil
}

func (c *Conn) readLength(short byte) (uint64, error) {
	switch short {
	case mediumLength:
		var b [2]byte
		_, err := io.ReadFull(c.r, b[:])

		return uint64(binary.BigEndian.Uint16(b[:])), err
	case longLength:
		var b [8]byte
		_, err := io.ReadFull(c.r, b[:])

		return binary.BigEndian.Uint64(b[:]), err
	default:
		return uint64(short), nil
	}
}

func (c *Conn) WriteText(payload []byte) error {
	return c.write(OpText, payload)
}

func (c *Conn) Ping() error {
	return c.write(opPing, nil)
}

func (c *Conn) write(op byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
		return err
	}

	_, err := c.conn.Write(frame(op, payload))

	return err
}

func frame(op byte, payload []byte) []byte {
	out := []byte{finBit | op}
	n := len(payload)

	switch {
	case n <= shortLength:
		out = append(out, maskBit|byte(n))
	case n <= 0xFFFF:
		out = append(out, maskBit|mediumLength)
		out = binary.BigEndian.AppendUint16(out, uint16(n))
	default:
		out = append(out, maskBit|longLength)
		out = binary.BigEndian.AppendUint64(out, uint64(n))
	}

	var mask [4]byte
	rand.Read(mask[:])

	masked := append([]byte(nil), payload...)
	applyMask(masked, mask)

	return append(append(out, mask[:]...), masked...)
}

func applyMask(b []byte, mask [4]byte) {
	for i := range b {
		b[i] ^= mask[i%4]
	}
}

func (c *Conn) Close() error {
	c.write(opClose, nil)

	return c.conn.Close()
}
