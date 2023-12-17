package main

type Message struct {
	Method  string      `json:"method"`
	Content interface{} `json:"args"`
}

type Response struct {
	Method  string      `json:"method"`
	Content interface{} `json:"response"`
}

type SentMessage struct {
	ID      string      `json:"id"`
	Content interface{} `json:"message"`
}

type Channel struct {
	LastMessages []interface{}
	Users        []*MutWS
}
