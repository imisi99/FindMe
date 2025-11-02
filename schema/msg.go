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
	CID     string
	Message []ViewMessage
}
