package concurrent_websocket

import (
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocket with synchronized reads and writes so that it may be used by multiple goroutines.
type ConcurrentWebSocket interface {
	WriteJSON(v interface{}) error                       // WriteJSON writes the JSON encoding of v as a message.
	WriteMessage(messageType int, data []byte) error     // WriteMessage is a helper method for getting a writer using NextWriter, writing the message and closing the writer.
	ReadJSON(v interface{}) error                        // ReadJSON reads the next JSON-encoded message from the connection and stores it in the value pointed to by v.
	ReadMessage() (messageType int, p []byte, err error) // ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
	RemoteAddr() net.Addr                                // RemoteAddr returns the remote network address.
	Close() error                                        // Close the websocket.
}

type GeneralWebSocketResponse struct {
	Op      string      `json:"op"`
	Payload interface{} `json:"payload"`
}

type concurrentWebSocketImpl struct {
	rlock sync.Mutex
	wlock sync.Mutex
	conn  *websocket.Conn
}

func NewConcurrentWebSocket(conn *websocket.Conn) ConcurrentWebSocket {
	return &concurrentWebSocketImpl{
		conn: conn,
	}
}

// WriteJSON writes the JSON encoding of v as a message.
func (w *concurrentWebSocketImpl) WriteJSON(v interface{}) error {
	// log.Printf("Preparing to write JSON message. Acquiring lock...\n")
	w.wlock.Lock()
	defer w.wlock.Unlock()
	// log.Printf("Preparing to write JSON message. Successfully acquired lock...\n")
	return w.conn.WriteJSON(v)
}

// Close the websocket.
func (w *concurrentWebSocketImpl) Close() error {
	// log.Printf("Preparing to close websocket. Acquiring lock...\n")
	w.wlock.Lock()
	defer w.wlock.Unlock()
	// log.Printf("Preparing to close websocket. Successfully acquired lock...\n")
	return w.conn.Close()
}

// WriteMessage is a helper method for getting a writer using NextWriter, writing the message and closing the writer.
func (w *concurrentWebSocketImpl) WriteMessage(messageType int, data []byte) error {
	// log.Printf("Preparing to write %v message. Acquiring lock...\n", messageType)
	w.wlock.Lock()
	defer w.wlock.Unlock()
	// log.Printf("Preparing to write %v message. Successfully acquired lock...\n", messageType)
	return w.conn.WriteMessage(messageType, data)
}

// ReadJSON reads the next JSON-encoded message from the connection and stores it in the value pointed to by v.
func (w *concurrentWebSocketImpl) ReadJSON(v interface{}) error {
	w.rlock.Lock()
	defer w.rlock.Unlock()

	return w.conn.ReadJSON(v)
}

// ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
func (w *concurrentWebSocketImpl) ReadMessage() (messageType int, p []byte, err error) {
	w.rlock.Lock()
	defer w.rlock.Unlock()

	return w.conn.ReadMessage()
}

// RemoteAddr returns the remote network address.
func (w *concurrentWebSocketImpl) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}
