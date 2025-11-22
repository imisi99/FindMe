package handlers

import (
	"net/http"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// TODO:
// Add a delete group chat endpoint where owners can delete the chat.

// DONE:
// A add user to chat endpoint for a project chat ?
// Select Friend to msg endpoint
// Close message endpoint
// It can just be modeled like discord where you can create and close msg so you can sort of start a chat by searching a friend ?
// An Endpoint for renaming chat and stuff.
// A way to name the group chats you can use before create.
// possibly if you search for the existing chat between users you can bring up group chat ?

// CreateMessage -> Add Message endpoint
func (s *Service) CreateMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.NewMessage
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var chat model.Chat
	if err := s.DB.FetchChat(payload.ChatID, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
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

	mesRes := schema.ViewMessage{
		ID:      msg.ID,
		Message: msg.Message,
		UserID:  uid,
		Sent:    msg.CreatedAt,
		Edited:  msg.UpdatedAt,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": mesRes})
}

// ViewMessages -> View chat history
func (s *Service) ViewMessages(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	cid := ctx.Query("id")
	if cid == "" {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Invalid chat id."})
		return
	}

	var chat model.Chat
	if err := s.DB.GetChatHistory(cid, &chat); err != nil {
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

// FetchUserChats -> Fetch all user chats.
func (s *Service) FetchUserChats(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadCM(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var chats []schema.ViewChat
	for _, chat := range user.Chats {
		var lastChat *model.UserMessage
		var chatName string
		if !chat.Group {
			if chat.Users[0].ID == uid {
				chatName = chat.Users[1].UserName
			} else {
				chatName = chat.Users[0].UserName
			}
		} else {
			chatName = chat.Name
		}
		if len(chat.Messages) > 0 {
			lastChat = chat.Messages[len(chat.Messages)-1]
			chats = append(chats, schema.ViewChat{
				Name: chatName,
				CID:  chat.ID,
				Message: []schema.ViewMessage{
					{
						ID:      lastChat.ID,
						Message: lastChat.Message,
						UserID:  lastChat.FromID,
						Sent:    lastChat.CreatedAt,
						Edited:  lastChat.UpdatedAt,
					},
				},
				Group: chat.Group,
			})
		} else {
			chats = append(chats, schema.ViewChat{
				Name:  chatName,
				CID:   chat.ID,
				Group: chat.Group,
			})
		}

	}

	ctx.JSON(http.StatusOK, gin.H{"msg": chats})
}

// EditMessage -> Edit a message endpoint
func (s *Service) EditMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.EditMessage
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
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

	msgRes := schema.ViewMessage{
		ID:      msg.ID,
		Message: msg.Message,
		UserID:  msg.FromID,
		Sent:    msg.CreatedAt,
		Edited:  msg.UpdatedAt,
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": msgRes})
}

// DeleteMessage -> Delete a message endpoint
func (s *Service) DeleteMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	mid := ctx.Query("id")
	if mid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid message id."})
		return
	}

	var msg model.UserMessage
	if err := s.DB.FetchMsg(&msg, mid); err != nil {
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

// OpenChat -> Endpoint for opening a chat between users with IDs.
func (s *Service) OpenChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	fid := ctx.Query("id")
	if fid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid friend id."})
		return
	}

	var chat model.Chat
	if err := s.DB.FindChat(uid, fid, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var msgs []schema.ViewMessage
	for _, msg := range chat.Messages {
		msgs = append(msgs, schema.ViewMessage{
			ID:      msg.ID,
			Message: msg.Message,
			UserID:  msg.FromID,
			Sent:    msg.CreatedAt,
			Edited:  msg.UpdatedAt,
		})
	}

	var chatName string
	if chat.Users[0].ID == uid {
		chatName = chat.Users[1].UserName
	} else {
		chatName = chat.Users[0].UserName
	}

	result := schema.ViewChat{
		Name:    chatName,
		CID:     chat.ID,
		Message: msgs,
		Group:   chat.Group,
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": result})
}

// RenameChat -> Rename the group chat endpoint
func (s *Service) RenameChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.RenameChat
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var chat model.Chat
	if err := s.DB.FetchChat(payload.ChatID, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if *chat.OwnerID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to rename this group chat."})
		return
	}

	chat.Name = payload.Name
	if err := s.DB.SaveChat(&chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Chat name updated successfully."})
}

// AddUserToChat -> Add a user to a chat endpoint
func (s *Service) AddUserToChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.AddUserChat
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var chat model.Chat
	if err := s.DB.FetchChat(payload.ChatID, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if *chat.OwnerID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You aren't permitted to add users to this chat."})
		return
	}

	if payload.UserID == uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You're already in the group."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.AddUserChat(&chat, &user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "User added to Chat."})
}

// RemoveUserChat -> Remove a user from the chat
func (s *Service) RemoveUserChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.AddUserChat
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse the payload."})
		return
	}

	var chat model.Chat

	if err := s.DB.FetchChat(payload.ChatID, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if *chat.OwnerID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You aren't permitted to remove users from this chat."})
		return
	}

	if payload.UserID == uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't remove yourself from the group."})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.RemoveUserChat(&chat, &user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// LeaveChat -> Leave a chat endpoint
func (s *Service) LeaveChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	cid := ctx.Query("id")
	if cid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid chat id."})
		return
	}

	var chat model.Chat
	if err := s.DB.FetchChat(cid, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var user model.User
	if err := s.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if err := s.DB.LeaveChat(&chat, &user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// DeleteChat -> Delete a group chat endpoint.
func (s *Service) DeleteChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if uid == "" || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	cid := ctx.Query("id")
	if cid == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid chat id."})
		return
	}

	var chat model.Chat
	if err := s.DB.FetchChat(cid, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if *chat.OwnerID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have the permission to delete this chat."})
		return
	}

	if err := s.DB.DeleteChat(&chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
