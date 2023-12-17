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
	jsoniter "github.com/json-iterator/go"
	"github.com/pccre/message-exchange/storage"
)

const greetingEnabled = true
const greeterUsername = "<color=red>System</color>"
const messageLengthLimit = 300
const messageLogLimit = 30

var wsConfig = websocket.Config{EnableCompression: true}
var json = jsoniter.ConfigFastest

var methodsList string
var pool = MutMap{Mut: &sync.RWMutex{}, Map: map[string]Channel{}}
var store storage.Storage = &storage.LocalStorage{Filename: "relationships.json"}

func isChat(channel string) bool {
	return channel == "PCC2.Main" || strings.HasPrefix(channel, "Creaty.PCC2.")
}

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

type MethodHandler func(c *MutWS, content interface{})

// fast, doesn't keep order
func remove[T any](s []T, i int) []T {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// keep order
func removeO[T any](s []T, i int) []T {
	return append(s[:i], s[i+1:]...)
}

func removeFromPool(channel string, c *MutWS) bool {
	ch := pool.Get(channel)
	for i, conn := range ch.Users {
		if conn == c {
			ch.Users = remove(ch.Users, i)
			pool.Set(channel, ch)
			return true
		}
	}
	return false
}

func BroadcastJSON(id string, content interface{}) {
	data, _ := json.Marshal(content)
	for _, c := range pool.Get(id).Users {
		c.WriteRaw(data)
	}
}

var methods = map[string]MethodHandler{
	"subscribe": func(c *MutWS, content interface{}) {
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
	"unsubscribe": func(c *MutWS, content interface{}) {
		channel, ok := content.(string)
		if !ok {
			c.WriteJSON(Response{Method: "unsubscribe", Content: "[ERROR] Channel name must be string"})
			return
		}

		if !removeFromPool(channel, c) {
			c.WriteJSON(Response{Method: "unsubscribe", Content: "[ERROR] Not subscribed to " + channel})
		}
	},
	"sendmessage": func(c *MutWS, content interface{}) {
		var id string
		var message map[string]interface{}
		var ch Channel
		msg, ok := content.(map[string]interface{})
		if !ok {
			goto notOk
		}
		id, ok = msg["id"].(string)
		if !ok {
			goto notOk
		}
		message, ok = msg["message"].(map[string]interface{})
		if !ok {
			goto notOk
		}

		if t, ok := message["text"].(string); ok && len(t) > messageLengthLimit {
			return
		}

		BroadcastJSON(id, Response{Method: "Message", Content: SentMessage{ID: id, Content: message}})

		if !isChat(id) {
			if _, ok := message["comand"].(string); ok {
				goto relationship
			}

			if _, ok := message["contact"].(map[string]interface{}); ok {
				goto relationship
			}
		}

		ch = pool.Get(id)
		ch.LastMessages = append(ch.LastMessages, message)
		if len(ch.LastMessages) > messageLogLimit {
			removeO(ch.LastMessages, 0)
		}
		pool.Set(id, ch)
		return
	notOk:
		c.WriteJSON(Response{Method: "sendmessage", Content: `[ERROR] Invalid content! You must pass JSON like this: {"method": "sendMessage", "args": {"id": "channel id", "message": {"any structure": "here"}}}`})
		return
	relationship:
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
		c := &MutWS{WS: cn, Mut: &sync.Mutex{}}
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

	}, wsConfig))

	log.Fatal(http.Listen(":8081"))
}
