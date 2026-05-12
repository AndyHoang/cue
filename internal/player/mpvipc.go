package player

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// mpvConn handles JSON-RPC communication with mpv over its IPC channel.
// The transport is a Unix domain socket on macOS/Linux and a named pipe on
// Windows; see mpvipc_unix.go / mpvipc_windows.go for the platform-specific
// dial logic and path helpers.
type mpvConn struct {
	conn net.Conn
	enc  *json.Encoder
	dec  *json.Decoder
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
