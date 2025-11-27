package handlers

import (
	"net/http"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// TODO:
// Broadcasting updated msgs

// WSChat -> Endpoint for websocket realtime chatting.
func (s *Service) WSChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	cid := ctx.Query("id")
	if !model.IsValidUUID(cid) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid chat id."})
		return
	}

	conn, err := upgrade.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to create a websocket connection."})
		return
	}

	client := &core.Client{
		Conn:     conn,
		UserID:   uid,
		ChatID:   cid,
		SendChan: make(chan *schema.ViewMessage),
	}

	s.Hub.Register <- client

	go client.ReadPump(s.Hub)
	go client.WritePump()
}
