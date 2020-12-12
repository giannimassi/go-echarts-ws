package ws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-echarts/go-echarts/v2/render"
	"github.com/gorilla/websocket"
)

// ScriptFmt is the template used for rendering ws-enabled charts
var ScriptFmt = `
<script type="text/javascript">
    let conn = new WebSocket("ws://%s/ws");
    conn.onclose = function(evt) {
        console.log("connection closed");
    }
    conn.onmessage = function(evt) {
        let data = JSON.parse(evt.data);
        goecharts_%s.setOption(data);
        console.log('Received data:', data);
    }
</script>
{{ end }}
`

// Render executed the Renderer Render function and adds the script
// required for updating data via websocket
func Render(w io.Writer, r render.Renderer, chartID, host string) error {
	var buf bytes.Buffer
	if err := r.Render(&buf); err != nil {
		return fmt.Errorf("while pre-rendering: %w", err)
	}

	script := fmt.Sprintf(ScriptFmt, host, chartID)
	_, err := w.Write(bytes.Replace(buf.Bytes(), []byte("</body>"), []byte(script), -1))
	return err
}

// Handler returns the ws handlers
// TODO: handle multiple subcriber
func Handler() (http.HandlerFunc, chan<- interface{}) {
	var dataC = make(chan interface{})
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				fmt.Println(err)
			}
			return
		}

		go writer(ws, dataC)
		reader(ws)
	}, dataC
}

func wsHandlerFunc(dataC <-chan interface{}) http.HandlerFunc {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				fmt.Println(err)
			}
			return
		}

		go writer(ws, dataC)
		reader(ws)
	}
}

const (
	// writeWait is the time allowed to write the file to the client.
	writeWait = 10 * time.Second
	// pongWait is the time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second
	// pingPeriod is the interval between pings sent to client. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn, dataC <-chan interface{}) {
	var pingTicker = time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		case changes := <-dataC:
			bytes, err := json.Marshal(changes)
			if err != nil {
				panic(fmt.Sprintf("unexpected err while marshalling changes: %v", changes))
			}
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, bytes); err != nil {
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
