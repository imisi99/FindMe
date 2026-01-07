package handlers

import (
	"net/http"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// TODO:
// Is there's a need to interact with the Chat Service here ?

// CreateMessage godoc
// @Summary     Sending of message to a chat
// @Description An endpoint for sending a message to a chat
// @Tags Msg
// @Accept json
// @Produce json
// @Param payload body schema.NewMessage true "Message payload"
// @Security BearerAuth
// @Success 201 {object} schema.DocMsgResponse "Message Sent"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/send-message [post]
func (s *Service) CreateMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
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

	ctx.JSON(http.StatusCreated, gin.H{"msg": mesRes})
}

// ViewMessages godoc
// @Summary    View All messages in a chat
// @Description An endpoint to view all messages in a chat (the chat history)
// @Tags Msg
// @Accept json
// @Produce json
// @Param id query string true "Chat ID"
// @Security BearerAuth
// @Success 200 {object} schema.DocViewChatHistory "Chat history"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/view-hist [get]
func (s *Service) ViewMessages(ctx *gin.Context) {
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

	var chat model.Chat
	if err := s.DB.GetChatHistory(cid, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}
	var hist schema.ViewChat
	hist.CID = cid
	hist.Group = chat.Group
	if hist.Group {
		hist.Name = chat.Name
	} else {
		if chat.Users[0].ID == uid {
			hist.Name = chat.Users[1].UserName
		} else {
			hist.Name = chat.Users[0].UserName
		}
	}

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

// FetchUserChats godoc
// @Summary    Fetch the current user chats
// @Description An endpoint for fetching all chats of the current user
// @Tags Msg
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocViewAllChats "User chats"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/view-chats [get]
func (s *Service) FetchUserChats(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
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

// EditMessage godoc
// @Summary    Editing a sent message
// @Description An endpoint for editing a sent message
// @Tags Msg
// @Accept json
// @Produce json
// @Param payload body schema.EditMessage true "Message payload"
// @Security BearerAuth
// @Success 202 {object} schema.DocMsgResponse "Message edited"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/edit-message [patch]
func (s *Service) EditMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
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

// DeleteMessage godoc
// @Summary     Delete a sent message
// @Description An endpoint for deleting a sent message of the current user
// @Tags Msg
// @Accept json
// @Produce json
// @Param id query string true "Msg ID"
// @Security BearerAuth
// @Success 204 {object} nil "Message deleted"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/delete-message [delete]
func (s *Service) DeleteMessage(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	mid := ctx.Query("id")
	if !model.IsValidUUID(mid) {
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

// OpenChat godoc
// @Summary     Open a chat between users
// @Description An endpoint for opening a chat between users with IDs.
// @Tags Msg
// @Accept json
// @Produce json
// @Param id query string true "User ID"
// @Security BearerAuth
// @Success 200 {object} schema.DocViewChatHistory "Chat opened"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/open-chat [get]
func (s *Service) OpenChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	fid := ctx.Query("id")
	if !model.IsValidUUID(fid) {
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

// RenameChat godoc
// @Summary   Renaming a group chat
// @Description An endpoint for renaming a group chat
// @Tags Msg
// @Accept json
// @Produce json
// @Param payload body schema.RenameChat true "Chat payload"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "Chat Updated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/rename-chat [patch]
func (s *Service) RenameChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
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

// AddUserToChat godoc
// @Summary Add a user to a group chat
// @Description An endpoint for adding users to a group chat
// @Tags Msg
// @Accept json
// @Produce json
// @Param payload body schema.AddUserChat true "Chat payload"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "User Added"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/add-user [put]
func (s *Service) AddUserToChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
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
	if err := s.DB.FetchUser(&user, payload.UserID); err != nil {
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

// RemoveUserChat godoc
// @Summary Remove a user from a group
// @Description An endpoint for removing a user from a group chat
// @Tags Msg
// @Accept json
// @Produce json
// @Param payload body schema.AddUserChat true "Chat payload"
// @Security BearerAuth
// @Success 204 {object} nil "User removed"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/remove-user [delete]
func (s *Service) RemoveUserChat(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
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
	if err := s.DB.FetchUser(&user, payload.UserID); err != nil {
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

// TransferOwner godoc
// @Summary Transfer group chat ownership to another user
// @Description An endpoint for transferring the ownership of a group chat to another user
// @Tags Msg
// @Accept json
// @Produce json
// @Param payload body schema.AddUserChat true "payload"
// @Security BearerAuth
// @Success 202 {object} schema.DocNormalResponse "Ownership Transferred"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Invalid payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/transfer-owner [patch]
func (s *Service) TransferOwner(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var payload schema.AddUserChat
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse payload."})
		return
	}

	var chat model.Chat
	if err := s.DB.FetchChat(payload.ChatID, &chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if !chat.Group {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can only transfer ownership in group chats."})
		return
	}

	if chat.OwnerID == nil || *chat.OwnerID != uid {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You don't have permission to transfer ownership."})
		return
	}

	ownerID := payload.UserID
	chat.OwnerID = &ownerID
	if err := s.DB.SaveChat(&chat); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{"msg": "Ownership transferred successfully."})
}

// LeaveChat godoc
// @Summary Leave a group chat
// @Description An endpoint for leaving a group chat
// @Tags Msg
// @Accept json
// @Produce json
// @Param id query string true "Chat ID"
// @Security BearerAuth
// @Success 204 {object} nil "Chat removed"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/leave-chat [delete]
func (s *Service) LeaveChat(ctx *gin.Context) {
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

	if *chat.OwnerID == user.ID {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "You can't leave this chat you can to delete it if you must or transfer ownership."})
		return
	}

	if err := s.DB.LeaveChat(&chat, &user); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// DeleteChat godoc
// @Summary Delete a group chat
// @Description An endpoint for deleting a group chat owned by the current user
// @Tags Msg
// @Accept json
// @Produce json
// @Param id query string true "chat ID"
// @Security BearerAuth
// @Success 204 {object} nil "Chat deleted"
// @Failure 400 {object} schema.DocNormalResponse "Invalid id"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 403 {object} schema.DocNormalResponse "Permission denied"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/msg/delete-chat [delete]
func (s *Service) DeleteChat(ctx *gin.Context) {
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
