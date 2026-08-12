package go2waa

import (
	"encoding/binary"
	"fmt"
	"io"
)

// CGI protocol commands
const (
	CmdConnect       int32 = 0   // First connection from the gateway
	CmdDisconnect    int32 = 1   // Last command from the server
	CmdReady         int32 = 2   // Server ready
	CmdAck           int32 = 3   // Positive acknowledgement
	CmdNak           int32 = 4   // Negative acknowledgement
	CmdGetEnv        int32 = 100 // Get environment variable
	CmdPutEnv        int32 = 110 // Response to GetEnv
	CmdGetVar        int32 = 200 // Get form variable
	CmdPutVar        int32 = 210 // Response to GetVar
	CmdPut           int32 = 201 // Write HTML output
	CmdGetAllVars    int32 = 300 // Get all variables
	CmdPutAllVars    int32 = 310 // Response to GetAllVars
	CmdOpenFile      int32 = 400 // Open file for writing
	CmdPutFileData   int32 = 401 // Write data to file
	CmdCloseFile     int32 = 402 // Close file
	CmdGetDocRoot    int32 = 500 // Get document root
	CmdGetScriptName int32 = 600 // Get script name
	CmdEmail         int32 = 700 // Send email
)

// Message represents a WAA protocol message
type Message struct {
	Cmd  int32
	Data []byte
}

// GetMessage reads a message from the WAA transport
// Format: <msgLen 4 bytes><cmd 4 bytes><data>
func GetMessage(conn io.Reader) (*Message, error) {
	// Read header (8 bytes: 4 for length + 4 for cmd)
	header := make([]byte, 8)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, fmt.Errorf("error reading header: %w", err)
	}

	msgLen := binary.LittleEndian.Uint32(header[0:4])
	cmd := int32(binary.LittleEndian.Uint32(header[4:8]))

	// msgLen includes the cmd (4 bytes), so data is msgLen - 4
	dataLen := msgLen - 4

	var data []byte
	if dataLen > 0 {
		data = make([]byte, dataLen)
		_, err = io.ReadFull(conn, data)
		if err != nil {
			return nil, fmt.Errorf("error reading data: %w", err)
		}
	}

	return &Message{Cmd: cmd, Data: data}, nil
}

// PutMessage sends a message to the WAA transport
// Format: <msgLen 4 bytes><cmd 4 bytes><data>
func PutMessage(conn io.Writer, cmd int32, data []byte) error {
	dataLen := 0
	if data != nil {
		dataLen = len(data)
	}

	// msgLen = 4 (cmd) + len(data)
	msgLen := uint32(4 + dataLen)

	// Build the complete message
	buf := make([]byte, 8+dataLen)
	binary.LittleEndian.PutUint32(buf[0:4], msgLen)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(cmd))
	if dataLen > 0 {
		copy(buf[8:], data)
	}

	_, err := conn.Write(buf)
	if err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}

	return nil
}

// PutMessageString sends a message with string data
func PutMessageString(conn io.Writer, cmd int32, data string) error {
	if data == "" {
		return PutMessage(conn, cmd, nil)
	}
	return PutMessage(conn, cmd, []byte(data))
}

// DataString returns the message data as a string
func (m *Message) DataString() string {
	if m.Data == nil {
		return ""
	}
	return string(m.Data)
}
