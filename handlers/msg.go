package handlers

import (
	"findme/database"
	"findme/model"
	"findme/schema"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Add Message endpoint
func CreateMessage(ctx *gin.Context) {
	db := database.GetDB()

	uid, tp := ctx.GetUint("userID"), ctx.GetString("purpose")
	if uid == 0 || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized user."})
		return
	}

	var user model.User
	if err := db.Preload("Friends.RecMessage").Where("id = ?", uid).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var payload schema.NewMessage
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Failed to parse payload."})
		return
	}

	var friend *model.User
	for _, fr := range user.Friends {		// This is not efficient i guess looping through all the user friends ? 
		if fr.UserName == payload.To {
			friend = fr
			break
		}
	}

	if friend == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	msg := model.UserMessage{
		From: user.ID,
		To: friend.ID,
		Message: payload.Message,
	}

	if err := db.Create(&msg).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send message."})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"message": "message sent successfully"})
}