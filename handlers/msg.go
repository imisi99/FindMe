package handlers

import (
	"net/http"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// TODO:
// Use the chat ID to find the friends instead of looping through user Friends
// Add a way to get the chatID if it's not present from the payload

// CreateMessage -> Add Message endpoint
func (s *Service) CreateMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.NewMessage
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse msg payload."})
		return
	}

	msg := model.UserMessage{
		ChatID:  payload.ChatID,
		Message: payload.Message,
		FromID:  uid,
	}

	if err := s.DB.AddMessage(&msg); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": msg})
}

// ViewMessages -> View messages history from a friend endpoint
func (s *Service) ViewMessages(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.GetChat
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload"})
		return
	}

	var chat model.Chat
	chatID := payload.ID
	if err := s.DB.GetChatHistory(chatID, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var hist schema.ViewChat
	for _, msg := range chat.Messages {
		hist.Message = append(hist.Message, schema.ViewMessage{
			ID:      msg.ID,
			UserID:  uid,
			Sent:    msg.CreatedAt,
			Edited:  msg.UpdatedAt,
			Message: msg.Message,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": hist})
}

// EditMessage -> Edit a message endpoint
func (s *Service) EditMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.EditMessage
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var msg model.UserMessage
	if err := s.DB.FetchMsg(&msg, payload.ID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if msg.FromID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You cannot delete a message that's not owned by you."})
		return
	}

	msg.Message = payload.Message
	if err := s.DB.SaveMsg(&msg); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": msg})
}

// DeleteMessage -> Delete a message endpoint
func (s *Service) DeleteMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.DeleteMessage
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var msg model.UserMessage
	if err := s.DB.FetchMsg(&msg, payload.ID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if msg.FromID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You cannot delete a message that's not owned by you."})
	}

	if err := s.DB.DeleteMsg(&msg); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
