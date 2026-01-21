package handlers

import (
	"net/http"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// GetTransactions godoc
// @Summary   Retrieves the transactions for a user
// @Description An endpoint that retrieves all the transactions of the current user
// @Tags Transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocTranscResposne "Transactions retrieved"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/transc/view [get]
func (s *Service) GetTransactions(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := s.DB.FetchUserPreloadT(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var response []schema.TransactionResponse

	for _, transc := range user.Transactions {
		response = append(response, schema.TransactionResponse{
			ID:      transc.ID,
			Amount:  transc.Amount,
			Channel: transc.Channel,
			Status:  transc.Status,
			PaidAt:  *transc.PaidAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"transactions": response})
}

// CreateTransaction godoc
// @Summary Creates a transaction for subscription payment
// @Description An endpoint for creating a transaction to initiate the payment process
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
func (s *Service) CreateTransaction(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var transc model.Transactions

	if err := s.DB.AddTransaction(&transc); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"msg": "ok"})
}
