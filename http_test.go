package showandtell

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketBus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pres := &Presentation{}
	serverAddr := "127.0.0.1:45369"
	server, err := NewPresentationServer(ctx, pres, "./test_project/slides", serverAddr)
	require.NoError(t, err)
	server.Run()
	defer server.Close()

	doneSubChan := make(chan bool, 1)

	go func() {
		conn, _, err := websocket.DefaultDialer.Dial("ws://"+serverAddr+"/messagebus", nil)
		require.NoError(t, err)
		err = conn.WriteJSON(WebSocketBusMessage{Type: "subscribe", Topic: "/foo/bar"})
		require.NoError(t, err)

		conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		doneSubChan <- true
		msg := &WebSocketBusMessage{}
		err = conn.ReadJSON(msg)
		require.NoError(t, err)
		assert.Equal(t, "message", msg.Type)
		assert.Equal(t, "/foo/bar", msg.Topic)
		conn.Close()
		cancel()
	}()
	timeOut := time.Tick(time.Second * 5)

	select {
	case <-timeOut:
		t.FailNow()
	case <-ctx.Done():
		t.FailNow()
	case <-doneSubChan:
	}
	// Give the message bus a little bit of time for its asynchronous operation
	time.Sleep(time.Millisecond * 50)
	server.centralBus.Publish("/foo/bar", json.RawMessage(`{"foo":"bar"}`))

	select {
	case <-timeOut:
		t.Fail()
	case <-ctx.Done():
	}

}
