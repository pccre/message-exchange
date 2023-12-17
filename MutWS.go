package main

import (
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
  return c.WriteRaw(data)
}

func (c *MutWS) WriteRaw(content []byte) error {
  c.Mut.Lock()
  defer c.Mut.Unlock()
  return c.WS.WriteMessage(1, content)
}
