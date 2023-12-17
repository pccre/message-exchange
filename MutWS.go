package main

import (
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type MutWS struct {
	WS  *websocket.Conn
	Mut *sync.Mutex
}

func (c *MutWS) WriteJSON(content interface{}) error {
	data, err := json.Marshal(content)
	if err != nil {
		return err
	}
	c.Mut.Lock()
	defer c.Mut.Unlock()
	log.Println(string(data))
	return c.WS.WriteMessage(1, data)
}
