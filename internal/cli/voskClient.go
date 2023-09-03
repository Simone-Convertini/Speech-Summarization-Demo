package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sync"

	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/models"
	"github.com/enriquebris/goconcurrentqueue"
	"github.com/gorilla/websocket"
	"github.com/minio/minio-go/v7"
)

type VoskWs struct {
	Host         string
	Port         string
	WriteChannel *chan models.QueueMessage
	wsConnection *websocket.Conn
}

var (
	connLock sync.Mutex
)

// Perform the dial to ws and returns the connection. Singleton implementation
func (vws *VoskWs) getVoskWSConnection() (*websocket.Conn, error) {
	connLock.Lock()
	defer connLock.Unlock()

	if vws.wsConnection != nil {
		return vws.wsConnection, nil
	}

	url := url.URL{Scheme: "ws", Host: vws.Host + ":" + vws.Port}
	wsConnection, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	vws.wsConnection = wsConnection
	if err != nil {
		return nil, err
	}

	// Define sample rate, some models require this configuration
	err = wsConnection.WriteMessage(websocket.TextMessage, []byte("{\"config\" : {\"sample_rate\": 16000 }}"))
	if err != nil {
		return nil, err
	}

	return wsConnection, nil
}

func (vws *VoskWs) CloseConnection() error {

	conn, err := vws.getVoskWSConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}
	vws.wsConnection = nil

	return nil
}

// To be used as a Goroutine to handle a file transcription
func ScribeParallel(file *minio.Object, channelVosk *chan string, wsClients ...*VoskWs) {
	textMap := sync.Map{}
	textQueue := goconcurrentqueue.NewFIFO()
	index := 0
	endNotification := make([]bool, 0)

	go func() {
		for {
			top, _ := textQueue.Get(0)
			text, ok := textMap.Load(top)

			if textQueue.GetLen() == 0 && len(endNotification) == len(wsClients) {
				close(*channelVosk)
				break
			}
			if text != nil && ok {
				*channelVosk <- text.(string)
				textQueue.Dequeue()
			}
		}
	}()

	for _, cli := range wsClients {
		go updateMap(cli, &textMap, &endNotification)
	}

	for {
		currentCli := wsClients[index%len(wsClients)]

		// Building the buffer sent in ws payload message
		buffer := make([]byte, 1000000)

		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Println(err)
		}

		// Stop sending buffer when the reading is complete
		if n == 0 && err == io.EOF {
			for _, cli := range wsClients {
				close(*cli.WriteChannel)
			}
			break
		}

		queueMessage := models.QueueMessage{Index: index, Buffer: buffer}
		*currentCli.WriteChannel <- queueMessage
		textQueue.Enqueue(index)
		index++
	}
}

func updateMap(wsClient *VoskWs, textMap *sync.Map, endNotification *[]bool) {
	defer wsClient.CloseConnection()

	for mes := range *wsClient.WriteChannel {
		conn, err := wsClient.getVoskWSConnection()
		if err != nil {
			fmt.Println(err)
		}

		err = conn.WriteMessage(websocket.BinaryMessage, mes.Buffer)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Read message from server
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Unmarshalling received message
		var m models.Message
		err = json.Unmarshal(msg, &m)
		if err != nil {
			fmt.Println(err)
			return
		}

		textMap.Store(mes.Index, m.Text)
	}
	*endNotification = append(*endNotification, true)
}
