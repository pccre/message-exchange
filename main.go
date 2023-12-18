package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/earlydata"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/mitchellh/mapstructure"
	"github.com/pccre/message-exchange/storage"
	"github.com/pccre/utils/Mut"
	"github.com/pccre/utils/c"
	"github.com/pccre/utils/shared"
)

// FEEL FREE TO CONFIGURE
// Greeting
const greetingEnabled bool = true
const greeterUsername string = "<color=red>System</color>"

// Limits
const messageLengthLimit int = 300
const messageLogLimit int = 30

// Storage
var store storage.Storage = &storage.LocalStorage{Filename: "relationships.json"}

// Advanced Greeting Configuration
func makeGreeting(channel string) map[string]interface{} {
	r := map[string]interface{}{}
	if channel == "PCC2.Main" {
		r["userName"] = greeterUsername
		r["image"] = ""
		r["id"] = "system"
	} else {
		r["nickname"] = greeterUsername
	}

	r["text"] = fmt.Sprintf("Welcome to OpenPCC Chat!\nThere are %d people in this channel.", len(pool.Map[channel].Users))
	r["avatar"] = "0.0.0.0.0.0.0"
	r["sender"] = "reserved"
	r["isPremium"] = false
	return r
}

// CODE STARTS HERE

var json = c.JSON
var methodsList string
var pool = Mut.Map[string, Channel]{Mut: &sync.RWMutex{}, Map: map[string]Channel{}}

func isChat(channel string) bool {
	return channel == "PCC2.Main" || strings.HasPrefix(channel, "Creaty.PCC2.")
}

type MethodHandler func(c *Mut.WS, content interface{})

func removeFromPool(channel string, c *Mut.WS) bool {
	ch := pool.Get(channel)
	for i, conn := range ch.Users {
		if conn == c {
			ch.Users = shared.Remove(ch.Users, i)
			pool.Set(channel, ch)
			return true
		}
	}
	return false
}

func ValidateMessage(channel string, content interface{}) (bool, MessageType) {
	var decoder *mapstructure.Decoder
	var err error

	decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, ErrorUnused: true, Result: &TradingChanMessage{}})
	if err == nil && decoder.Decode(content) == nil {
		return true, NormalMessage
	}

	decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, ErrorUnused: true, Result: &MainChanMessage{}})
	if err == nil && decoder.Decode(content) == nil {
		return true, NormalMessage
	}

	if !isChat(channel) {
		decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, ErrorUnused: true, Result: &CreateContactTrading{}})
		if err == nil && decoder.Decode(content) == nil {
			return true, RelationshipMessage
		}

		decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, ErrorUnused: true, Result: &CreateContactMain{}})
		if err == nil && decoder.Decode(content) == nil {
			return true, RelationshipMessage
		}
	}
	return false, 0
}

func SendTo(id string, content interface{}) {
	BroadcastJSON(id, Response{Method: "Message", Content: SentMessage{ID: id, Content: content}})
}

func BroadcastJSON(id string, content interface{}) {
	data, _ := json.Marshal(content)
	for _, c := range pool.Get(id).Users {
		c.WriteRaw(data)
	}
}

var methods = map[string]MethodHandler{
	"subscribe": func(c *Mut.WS, content interface{}) {
		channel, ok := content.(string)
		if !ok {
			c.WriteJSON(Response{Method: "subscribe", Content: "[ERROR] Channel name must be string"})
			return
		}

		ch := pool.Get(channel)
		for _, conn := range ch.Users {
			if conn == c {
				c.WriteJSON(Response{Method: "subscribe", Content: "[ERROR] You are already subscribed to " + channel})
				return
			}
		}

		ch.Users = append(ch.Users, c)
		pool.Set(channel, ch)
		for _, msg := range ch.LastMessages {
			c.WriteJSON(Response{Method: "Message", Content: SentMessage{ID: channel, Content: msg}})
		}
		if isChat(channel) {
			if greetingEnabled {
				c.WriteJSON(Response{Method: "Message", Content: SentMessage{ID: channel, Content: makeGreeting(channel)}})
			}
		} else {
			st, err := store.GetRelationships(channel)
			if err != nil {
				return
			}

			for rel := range st {
				c.WriteJSON(Response{Method: "Message", Content: SentMessage{ID: channel, Content: rel}})
			}
		}
	},
	"unsubscribe": func(c *Mut.WS, content interface{}) {
		channel, ok := content.(string)
		if !ok {
			c.WriteJSON(Response{Method: "unsubscribe", Content: "[ERROR] Channel name must be string"})
			return
		}

		if !removeFromPool(channel, c) {
			c.WriteJSON(Response{Method: "unsubscribe", Content: "[ERROR] Not subscribed to " + channel})
		}
	},
	"sendmessage": func(c *Mut.WS, content interface{}) {
		var id string
		var message map[string]interface{}
		var ch Channel
		var ok bool
		var msgtype MessageType

		message, ok = content.(map[string]interface{})
		if !ok {
			goto notOk
		}

		id, ok = message["id"].(string)
		if !ok {
			goto notOk
		}

		message, ok = message["message"].(map[string]interface{})
		if !ok {
			goto notOk
		}

		switch ok, msgtype = ValidateMessage(id, message); true {
		case !ok:
			goto notOk
		case msgtype == RelationshipMessage:
			goto relationship
		}

		if t := message["text"].(string); len(t) > messageLengthLimit {
			return
		}

		SendTo(id, message)
		ch = pool.Get(id)
		ch.LastMessages = append(ch.LastMessages, message)
		if len(ch.LastMessages) > messageLogLimit {
			shared.RemoveO(ch.LastMessages, 0)
		}
		pool.Set(id, ch)
		return
	notOk:
		c.WriteJSON(Response{Method: "sendmessage", Content: `[ERROR] Invalid content! You must pass JSON like this: {"method": "sendMessage", "args": {"id": "channel id", "message": {"message structure": "here"}}}`})
		return
	relationship:
		SendTo(id, message)
		store.AddRelationship(id, message)
	},
}

func main() {
	go func() {
		store.Load()
		for method := range methods {
			methodsList += method + ", "
		}
		methodsList = methodsList[:len(methodsList)-2]
	}()
	http := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
		GETOnly:     true,
	})

	http.Use(recover.New())
	http.Use(earlydata.New())

	http.Use("/Chat", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	http.Get("/Chat", websocket.New(func(cn *websocket.Conn) {
		c := &Mut.WS{WS: cn, Mut: &sync.Mutex{}}
		var (
			msg    []byte
			parsed Message
			err    error
			found  bool
		)

		for {
			if _, msg, err = c.WS.ReadMessage(); err != nil {
				log.Println("read err:", err)
				for key := range pool.Map {
					removeFromPool(key, c)
				}
				return
			}

			err = json.Unmarshal(msg, &parsed)
			if err != nil {
				c.WriteJSON(Response{Method: "OnMessage", Content: `[ERROR] Invalid content! You must pass JSON like this: {"method": "methodName", "args": "arguments"}`})
				continue
			}

			parsed.Method = strings.ToLower(parsed.Method)

			found = false
			for method, handler := range methods {
				if method == parsed.Method {
					go handler(c, parsed.Content)
					found = true
					break
				}
			}

			if !found {
				c.WriteJSON(Response{Method: "OnMessage", Content: "[ERROR] Invalid method! Method list: " + methodsList})
				continue
			}
		}

	}, c.WSConfig))

	log.Fatal(http.Listen(":8081"))
}
