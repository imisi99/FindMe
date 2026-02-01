package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// TODO: Check on the Closing of the resp for keeping alive requests

// TODO:
// Create Plan and add plan to DB
// Add Sub Code for paystack to user Model and Sub plan to Sub model from plan Model
// Notify user with the Cancel Sub, Failed Transaction, Ending Sub soon email
// Augmented Data for user (card ending , date and month, card type, next payment date)

// DONE:
// Write email for the successful / failed surcharge for Subscriptions
// Cancel sub event and also endpoint(if it uses one)

type Transc interface {
	GetTransactions(ctx *gin.Context)
	InitializeTransaction(ctx *gin.Context)
	UpdateSubscriptionCard(ctx *gin.Context)
	CancelSubscription(ctx *gin.Context)
	EnableSubscription(ctx *gin.Context)
}

type TranscService struct {
	Email     core.Email
	DB        core.DB
	SecretKey string
	Client    *http.Client
}

func NewTranscService(db core.DB, secret string, client *http.Client) *TranscService {
	return &TranscService{DB: db, SecretKey: secret, Client: client}
}

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
func (t *TranscService) GetTransactions(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := t.DB.FetchUserPreloadT(&user, uid); err != nil {
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

// InitializeTransaction godoc
// @Summary An endpoint for initializing a transaction on paystack
// @Description An endpoint for initializing a transaction on paystack to receive a checkout url for payment
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param amount query string true "amount"
// @Param amount query string true "plan"
// @Success 200 {object} schema.DocInitTranscResponse "Success"
// @Failure 400 {object} schema.DocNormalResponse "Bad Query"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "Bad Gateway"
// @Router /api/transc/initialize [get]
func (t *TranscService) InitializeTransaction(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	amount, plan := ctx.Query("amount"), ctx.Query("plan")
	if amount == "" || plan == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Transaction amount or plan not in query."})
		return
	}

	var user model.User
	if err := t.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	payload := map[string]string{
		"email":  user.Email,
		"amount": amount,
		"plan":   plan,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "https://api.paystack.co/transactions/initialize", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+t.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)

	if err != nil || resp.StatusCode != http.StatusOK {
		log.Println("[TRANSACTION] Failed to initialize paystack transaction, err -> ", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to initialize paystack transaction"})
		return
	}

	defer resp.Body.Close()

	var paystack schema.InitTransaction

	if err := json.NewDecoder(resp.Body).Decode(&paystack); err != nil {
		log.Println("[TRANSACTION] Failed to parse the response of the paystack init transaction, err -> ", err)
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse paystack response"})
		return
	}

	if !paystack.Status || paystack.Data.AuthorizationURL == "" || paystack.Data.AccessCode == "" || paystack.Data.Reference == "" {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Unable to retrieve checkout url or access code from paystack"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"uri": paystack.Data.AuthorizationURL, "token": paystack.Data.AccessCode})
}

// VerifyTranscWebhook godoc
func (t *TranscService) VerifyTranscWebhook(ctx *gin.Context) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	signature := ctx.GetHeader("x-paystack-signature")

	mac := hmac.New(sha512.New, []byte(t.SecretKey))
	mac.Write(body)
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(signature)) {
		ctx.Status(http.StatusUnauthorized)
		return
	}

	var event schema.PaystackEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Println("[TRANSACTION] An error occured in the webhook for the transaction while parsing payload, err -> ", err.Error())
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}

	var user model.User
	if err := t.DB.FetchUser(&user, event.Data.Customer.Email); err != nil {
		log.Println("[TRANSACTION] Failed to complete transaction as the customer could not be identified, err -> ", err.Error())
		ctx.Status(http.StatusUnauthorized)
		return
	}

	var transc model.Transactions
	switch event.Event {
	case model.PaystackChargeSuccess:
		if err := t.DB.FetchTransaction(event.Data.Reference, &transc); err != nil {
			cm := err.(*core.CustomMessage)
			if cm.Code == http.StatusNotFound {
				amount, _ := strconv.ParseInt(event.Data.Amount, 10, 64)

				transc.UserID = user.ID
				transc.PaystackRef = event.Data.Reference
				transc.Amount = amount
				transc.Status = event.Data.Status
				transc.Channel = event.Data.Channel
				transc.Curency = event.Data.Currency
				transc.PaidAt = &event.Data.PaidAt

				sub := model.Subscriptions{
					UserID:    user.ID,
					Status:    model.StatusActive,
					StartDate: time.Now(),
					EndDate:   time.Now().Add(time.Hour * 24 * 30),
				}

				user.LastSubEnd = &sub.EndDate
				user.RecurringSub = true

				if err := t.DB.AddTranscSub(&transc, &sub, &user); err != nil {
					ctx.Status(http.StatusInternalServerError)
					return
				}

			} else {
				ctx.Status(cm.Code)
				return
			}
		} else {
			transc.Status = event.Data.Status
			transc.Channel = event.Data.Channel
			transc.Curency = event.Data.Currency
			transc.PaidAt = &event.Data.PaidAt

			sub := model.Subscriptions{
				TransactionID: transc.ID,
				UserID:        user.ID,
				Status:        model.StatusActive,
				StartDate:     time.Now(),
				EndDate:       time.Now().Add(time.Hour * 24 * 30),
			}

			user.LastSubEnd = &sub.EndDate
			user.RecurringSub = true

			if err := t.DB.SaveTranscAddSub(&transc, &sub, &user); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}
		}
		ctx.Status(http.StatusOK)
		return
	case model.PaystackInvoiceUpdate:
		if !event.Data.Paid {
			sub := model.Subscriptions{
				UserID:    user.ID,
				Status:    model.StatusFailed,
				StartDate: time.Now(),
				EndDate:   time.Now().Add(time.Hour * 24 * 30),
			}

			grace := user.LastSubEnd.Add(time.Hour * 24 * 7)
			user.LastSubEnd = &grace

			if err := t.DB.AddFailedSub(&sub, &user); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}

			t.Email.QueueTransactionFailedEmail(user.UserName, event.Data.Amount, event.Data.Currency, event.Data.Plan.Name, "", user.Email)
		}
		ctx.Status(http.StatusOK)
		return
	case model.PaystackCancelSub:
		user.RecurringSub = false
		if err := t.DB.SaveUser(&user); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}
		t.Email.QueueCancelSubscription(user.UserName, event.Data.Plan.Name, *user.LastSubEnd, user.Email)
	default:
		ctx.Status(http.StatusOK)
		return
	}
}

// UpdateSubscriptionCard godoc
// @Summary This is an endpoint for udpating card details on paystack
// @Description This is an endpoint for retrieving link for updating card details used for transaction on paystack
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocUpdateCardSub "Link Generated"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "Failed communication with external service"
// @Router /api/transc/update-card [get]
func (t *TranscService) UpdateSubscriptionCard(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := t.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.paystack.co/subscription/"+*user.PaystackSubCode+"/manage/link", nil)
	req.Header.Set("Authorization", "Bearer "+t.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Println("[TRANSACTION] An error occured while trying to generate a update card link on paystack, err -> ", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to communicate with paystack"})
		return
	}

	defer resp.Body.Close()
	var card schema.PaystackUpdateCard

	body, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &card); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to Unmarshal payload from paystack, err -> ", err.Error())
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse response from paystack"})
		return
	}

	if !card.Status {
		ctx.JSON(resp.StatusCode, gin.H{"msg": card.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": card.Message, "link": card.Data.Link})
}

// CancelSubscription godoc
// @Summary An endpoint for canceling a subscription
// @Description An endpoint for canceling a subscription
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocNormalResponse "Success"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "Failed communication with external service"
// @Router /api/transc/cancel-sub [patch]
func (t *TranscService) CancelSubscription(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := t.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	payload := map[string]string{
		"code":  *user.PaystackSubCode,
		"token": *user.PaystackEmailToken,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "https://api.paystack.co/subscription/disable", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+t.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Println("[TRANSACTION] An error occured while trying to cancel a subscription, err -> ", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to communicate with paystack."})
		return
	}

	defer resp.Body.Close()
	var sub schema.PaystackSubResp

	if err := json.Unmarshal(body, &sub); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to Unmarshal payload from paystack, err -> ", err.Error())
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse response from paystack"})
		return
	}

	if !sub.Status {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": sub.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": sub.Message})
}

// EnableSubscription godoc
// @Summary An endpoint for enabling a subscription
// @Description An endpoint for re enabling a subscription
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocNormalResponse "Success"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 402 {object} schema.DocNormalResponse "Payment required"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "Failed communication with external service"
// @Router /api/transc/enable-sub [patch]
func (t *TranscService) EnableSubscription(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	var user model.User
	if err := t.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if time.Now().After(*user.LastSubEnd) {
		ctx.JSON(http.StatusPaymentRequired, gin.H{"msg": "Subscription has expired and needs to be renewed."})
		return
	}

	payload := map[string]string{
		"code":  *user.PaystackSubCode,
		"token": *user.PaystackEmailToken,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "https://api.paystack.co/subscription/enable", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+t.SecretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to communicate with paystack."})
		return
	}

	defer resp.Body.Close()
	var sub schema.PaystackSubResp

	if err := json.Unmarshal(body, &sub); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to Unmarshal payload from paystack, err -> ", err.Error())
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse response from paystack"})
		return
	}

	if !sub.Status {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": sub.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": sub.Message})
}
