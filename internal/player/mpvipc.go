package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// mpvConn handles JSON-RPC communication with mpv over a Unix socket.
type mpvConn struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
}

// dialMPV attempts to connect to the mpv IPC socket.
// It retries for up to 3 seconds as mpv takes a moment to create the socket.
func dialMPV(socketPath string) (*mpvConn, error) {
	var conn net.Conn
	var err error

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.Dial("unix", socketPath)
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to mpv IPC: %w", err)
	}

	return &mpvConn{
		conn: conn,
		enc:  json.NewEncoder(conn),
		dec:  json.NewDecoder(bufio.NewReader(conn)),
	}, nil
}

// GetTimePos queries the current playback position in seconds.
func (c *mpvConn) GetTimePos() (float64, error) {
	requestID := time.Now().UnixNano()
	req := map[string]interface{}{
		"command":    []interface{}{"get_property", "time-pos"},
		"request_id": requestID,
	}

	if err := c.enc.Encode(req); err != nil {
		return 0, err
	}

	for {
		var resp struct {
			Data      float64 `json:"data"`
			Error     string  `json:"error"`
			RequestID int64   `json:"request_id"`
			Event     string  `json:"event"`
		}

		if err := c.dec.Decode(&resp); err != nil {
			return 0, err
		}

		// Skip event messages
		if resp.Event != "" {
			continue
		}

		// Check if this is the response to our command
		if resp.RequestID == requestID {
			if resp.Error != "success" && resp.Error != "" {
				return 0, fmt.Errorf("mpv error: %s", resp.Error)
			}
			return resp.Data, nil
		}
	}
}

func (c *mpvConn) Close() error {
	return c.conn.Close()
}
