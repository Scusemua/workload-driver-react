package proxy

import (
	"context"
	"net"
	"time"

	"nhooyr.io/websocket"
)

// Reference:
// - https://github.com/pojntfx/go-app-grpc-chat-frontend-web/
//
// TODO(Ben): Update license accordingly.

type WebSocketProxyClient struct {
	timeout time.Duration
}

func NewWebSocketProxyClient(timeout time.Duration) *WebSocketProxyClient {
	client := &WebSocketProxyClient{
		timeout: timeout,
	}

	return client
}

// Pass this to the grpc.Dial function, wrapped in a grpc.WithContextDialer.
//
// /* Begin Example: */
//
// proxy := websocketproxy.NewWebSocketProxyClient(time.Minute)
//
// conn, err := grpc.Dial("ws://127.0.0.1:9090", grpc.WithContextDialer(proxy.Dialer), grpc.WithInsecure())
//
//	if err != nil {
//		panic(err)
//	}
//
// defer conn.Close()
//
// /* End Example */
func (p *WebSocketProxyClient) Dialer(ctx context.Context, url string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	return websocket.NetConn(context.Background(), conn, websocket.MessageBinary), nil
}
