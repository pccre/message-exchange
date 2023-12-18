package main

import "github.com/pccre/utils/Mut"

type Message struct {
	Method  string      `json:"method"`
	Content interface{} `json:"args"`
}

type Response struct {
	Method  string      `json:"method"`
	Content interface{} `json:"response"`
}

type SentMessage struct {
	ID      string      `json:"id" mapstructure:"id"`
	Content interface{} `json:"message" mapstructure:"message"`
}

type Channel struct {
	LastMessages []interface{}
	Users        []*Mut.WS
}

type BaseChanMessage struct {
	Text      string `mapstructure:"text" json:"text"`
	UserHash  string `mapstructure:"sender" json:"sender"`
	Avatar    string `mapstructure:"avatar" json:"avatar"`
	IsPremium bool   `mapstructure:"isPremium" json:"isPremium"`
}

type MainChanMessage struct {
	ID              string `mapstructure:"id" json:"id"`
	BaseChanMessage `mapstructure:",squash"`
	UserName        string `mapstructure:"userName" json:"userName"`
	Image           string `mapstructure:"image" json:"image"`
}

type TradingChanMessage struct {
	BaseChanMessage `mapstructure:",squash"`
	UserName        string `mapstructure:"nickname" json:"nickname"`
}

type Contact struct {
	UserHash  string `mapstructure:"userHash" json:"userHash"`
	Avatar    string `mapstructure:"avatar" json:"avatar"`
	UserName  string `mapstructure:"userName" json:"userName"`
	IsPremium bool   `mapstructure:"isPremium" json:"isPremium"`
}

type CreateContactMain struct {
	Contact Contact `mapstructure:"contact" json:"contact"`
}

type CreateContactTrading struct {
	Command  string `mapstructure:"comand" json:"comand"`
	UserHash string `mapstructure:"sender" json:"sender"`
	UserName string `mapstructure:"nickname" json:"nickname"`
	Avatar   string `mapstructure:"avatar" json:"avatar"`
}

type MessageType uint8

const (
	NormalMessage MessageType = iota
	RelationshipMessage
)
