// Package schema -> Schema for the app
package schema

import "time"

type NewMessage struct {
	Message string `json:"msg" binding:"required"`
	ChatID  string `json:"chat_id" binding:"omitempty"`
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

type GetChat struct {
	ID string `json:"chat_id" binding:"omitempty"`
}

type DeleteMessage struct {
	ID string `json:"msg_id" binding:"required"`
}

type ViewChat struct {
	ID      string
	Message []ViewMessage
}
