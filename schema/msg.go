// Package schema -> Schema for the app
package schema

import "time"

type NewMessage struct {
	Message string `json:"msg" binding:"required"`
	To      string `json:"user" binding:"required"`
}

type EditMessage struct {
	Message string `json:"msg" binding:"required"`
}

type ViewMessage struct {
	ID      uint
	Message string
	Sent    time.Time
	Edited  time.Time
}
