package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TODO:
// I Could use like a chat ID to reference a user and his chat ?
// This would require a Model ?
// It would make getting the chat history very fast though :)

// CreateMessage -> Add Message endpoint
func (m *Service) CreateMessage(ctx *gin.Context) {
	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := m.DB.Preload("Friends").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})
		}
		return
	}

	var payload schema.NewMessage
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	// This is not efficient i guess looping through all the user friends ?

	var friend *model.User
	for _, fr := range user.Friends {
		if fr.UserName == payload.To {
			friend = fr
			break
		}
	}

	if friend == nil {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "user is not your friend."})
		return
	}

	msg := model.UserMessage{
		FromID:  user.ID,
		ToID:    friend.ID,
		Message: payload.Message,
	}

	if err := m.DB.Create(&msg).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send message."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "message sent successfully."})
}

// ViewMessages -> View messages history from a friend endpoint
func (m *Service) ViewMessages(ctx *gin.Context) {
	uid, tp, username := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user, friend model.User
	if err := m.DB.Preload("RecMessage").Preload("Message").Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve user from db."})
		}
		return
	}

	if err := m.DB.Where("username = ?", username).First(&friend).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Friend not found."})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve friend from db."})
		}
		return
	}

	var hist []schema.ViewMessage
	for _, msg := range user.Message {
		if msg.ToID == friend.ID {
			hist = append(hist, schema.ViewMessage{
				ID:      msg.ID,
				Message: msg.Message,
				Sent:    msg.CreatedAt,
				Edited:  msg.UpdatedAt,
			})
		}
	}

	for _, msg := range user.RecMessage {
		if msg.FromID == friend.ID {
			hist = append(hist, schema.ViewMessage{
				ID:      msg.ID,
				Message: msg.Message,
				Sent:    msg.CreatedAt,
				Edited:  msg.UpdatedAt,
			})
		}
	}

	ctx.JSON(http.StatusOK, hist)
}

// EditMessage -> Edit a message endpoint
func (m *Service) EditMessage(ctx *gin.Context) {
	uid, tp, idStr := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid message id."})
		return
	}

	var msg model.UserMessage
	if err := m.DB.Where("id = ?", uint(id)).Where("from_id = ?", uid).First(&msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Message not found."})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve message from db."})
		}
		return
	}

	var payload schema.EditMessage
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	msg.Message = payload.Message

	if err := m.DB.Save(&msg).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to edit message."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Message Edited successfully."})
}

// DeleteMessage -> Delete a message endpoint
func (m *Service) DeleteMessage(ctx *gin.Context) {
	uid, tp, idStr := ctx.GetUint("userID"), ctx.GetString("purpose"), ctx.Query("id")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Invalid message id."})
		return
	}

	var msg model.UserMessage
	if err := m.DB.Where("id = ?", uint(id)).Where("from_id = ?", uid).First(&msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "Message not found."})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve message from db."})
		}
		return
	}

	if err := m.DB.Delete(&msg).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete message."})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
