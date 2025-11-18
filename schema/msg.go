// Package schema -> Schema for the app
package schema

import "time"

type NewMessage struct {
	Message string `json:"msg" binding:"required"`
	ChatID  string `json:"chat_id" binding:"required"`
}

type EditMessage struct {
	ID      string `json:"msg_id" binding:"required"`
	Message string `json:"msg" binding:"required"`
}

type ViewMessage struct {
	ID      string
	Message string
	UserID  string
	Sent    time.Time
	Edited  time.Time
}

type ViewChat struct {
	Name    string
	CID     string
	Message []ViewMessage
	Group   bool
}

type AddUserChat struct {
	ChatID string `json:"chat_id" binding:"required"`
	UserID string `json:"user_id" binding:"required"`
}
