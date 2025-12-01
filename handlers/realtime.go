package handlers

import (
	"net/http"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// WSChat godoc
// @Summary  A websocket message hub to simulate real-time chatting
// @Description An websocket endpoint that simulates a real-time chatting experience
// @Tags Msg
// @Accept json
// @Produce json
// @Param id query string true "Chat ID"
// @Security BearerAuth
// @Success
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "server error"
// @Router /api/msg/ws/chat [get]
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
