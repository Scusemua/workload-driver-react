package domain

import "net"

// WebSocket with synchronized reads and writes so that it may be used by multiple goroutines.
type ConcurrentWebSocket interface {
	WriteJSON(v interface{}) error                       // WriteJSON writes the JSON encoding of v as a message.
	WriteMessage(messageType int, data []byte) error     // WriteMessage is a helper method for getting a writer using NextWriter, writing the message and closing the writer.
	ReadJSON(v interface{}) error                        // ReadJSON reads the next JSON-encoded message from the connection and stores it in the value pointed to by v.
	ReadMessage() (messageType int, p []byte, err error) // ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
	RemoteAddr() net.Addr                                // RemoteAddr returns the remote network address.
}

type GeneralWebSocketResponse struct {
	Op      string      `json:"op"`
	Payload interface{} `json:"payload"`
}
