package concurrent_websocket

import (
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

type BasicConcurrentWebSocket struct {
	rlock sync.Mutex
	wlock sync.Mutex
	conn  *websocket.Conn

	metadata      map[string]interface{}
	metadataMutex sync.Mutex
}

func NewConcurrentWebSocket(conn *websocket.Conn) *BasicConcurrentWebSocket {
	return &BasicConcurrentWebSocket{
		conn:     conn,
		metadata: map[string]interface{}{},
	}
}

func (w *BasicConcurrentWebSocket) AddMetadata(key string, value interface{}) {
	w.metadataMutex.Lock()
	defer w.metadataMutex.Unlock()

	w.metadata[key] = value
}

func (w *BasicConcurrentWebSocket) GetMetadata(key string) (interface{}, bool) {
	w.metadataMutex.Lock()
	defer w.metadataMutex.Unlock()

	value, ok := w.metadata[key]
	return value, ok
}

// WriteJSON writes the JSON encoding of v as a message.
func (w *BasicConcurrentWebSocket) WriteJSON(v interface{}) error {
	// log.Printf("Preparing to write JSON message. Acquiring lock...\n")
	w.wlock.Lock()
	defer w.wlock.Unlock()
	// log.Printf("Preparing to write JSON message. Successfully acquired lock...\n")
	return w.conn.WriteJSON(v)
}

// Close the websocket.
func (w *BasicConcurrentWebSocket) Close() error {
	// log.Printf("Preparing to close websocket. Acquiring lock...\n")
	w.wlock.Lock()
	defer w.wlock.Unlock()
	// log.Printf("Preparing to close websocket. Successfully acquired lock...\n")
	return w.conn.Close()
}

// WriteMessage is a helper method for getting a writer using NextWriter, writing the message and closing the writer.
func (w *BasicConcurrentWebSocket) WriteMessage(messageType int, data []byte) error {
	// log.Printf("Preparing to write %v message. Acquiring lock...\n", messageType)
	w.wlock.Lock()
	defer w.wlock.Unlock()
	// log.Printf("Preparing to write %v message. Successfully acquired lock...\n", messageType)
	return w.conn.WriteMessage(messageType, data)
}

// ReadJSON reads the next JSON-encoded message from the connection and stores it in the value pointed to by v.
func (w *BasicConcurrentWebSocket) ReadJSON(v interface{}) error {
	w.rlock.Lock()
	defer w.rlock.Unlock()

	return w.conn.ReadJSON(v)
}

// ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
func (w *BasicConcurrentWebSocket) ReadMessage() (messageType int, p []byte, err error) {
	w.rlock.Lock()
	defer w.rlock.Unlock()

	return w.conn.ReadMessage()
}

// RemoteAddr returns the remote network address.
func (w *BasicConcurrentWebSocket) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}
