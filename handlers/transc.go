package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"findme/core"
	"findme/model"
	"findme/schema"

	"github.com/gin-gonic/gin"
)

// TODO: Check on the Closing of the resp for keeping alive requests

// DONE:
// Write email for the successful / failed surcharge for Subscriptions
// Cancel sub event and also endpoint(if it uses one)
// Notify user with the Cancel Sub, Failed Transaction
// Create Plan and add plan to RDB
// Augmented Data for user (card ending , date and month, card type, next payment date)
// Add Sub Code for paystack to user Model and Sub plan to Sub model from plan Model
//

type Transc interface {
	GetTransactions(ctx *gin.Context)
	InitializeTransaction(ctx *gin.Context)
	UpdateSubscriptionCard(ctx *gin.Context)
	CancelSubscription(ctx *gin.Context)
	EnableSubscription(ctx *gin.Context)
	ViewPlans(ctx *gin.Context)
	VerifyTranscWebhook(ctx *gin.Context)
}

type TranscService struct {
	Email     core.Email
	DB        core.DB
	RDB       core.Cache
	SecretKey string
	Client    *http.Client
}

func NewTranscService(db core.DB, rdb core.Cache, email core.Email, secret string, client *http.Client) *TranscService {
	return &TranscService{DB: db, RDB: rdb, Email: email, SecretKey: secret, Client: client}
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
// @Param plan query string true "plan"
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

	payload := map[string]any{
		"email":  user.Email,
		"amount": amount,
		"plan":   plan,
		// "channels": []string{"card"},
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "https://api.paystack.co/transaction/initialize", bytes.NewBuffer(body))
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

	data, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(data, &paystack); err != nil {
		log.Println("[TRANSACTION] Failed to unmarshal the response of the paystack init transaction, err -> ", err)
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse paystack response"})
	}

	if !paystack.Status || paystack.Data.AuthorizationURL == "" || paystack.Data.AccessCode == "" || paystack.Data.Reference == "" {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Unable to retrieve checkout url or access code from paystack"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"uri": paystack.Data.AuthorizationURL, "token": paystack.Data.AccessCode})
}

// VerifyTranscWebhook godoc
// @Summary This is a webhook for the paystack transaction events
// @Description An endpoint for intercepting the paystack webhooks transaction events
// @Tags Transaction
// @Accept json
// @Produce json
// @Success 200 {object} schema.DocNormalResponse "Success"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Router /api/transc/webhook [post]
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
	log.Println(string(body))
	if err := json.Unmarshal(body, &event); err != nil {
		log.Println("[TRANSACTION] An error occured in the webhook for the transaction while parsing payload, err -> ", err.Error())
		ctx.Status(http.StatusUnprocessableEntity)
		return
	}

	var user model.User
	if err := t.DB.SearchUserEmail(&user, event.Data.Customer.Email); err != nil {
		log.Println("[TRANSACTION] Failed to complete transaction as the customer could not be identified, err -> ", err.Error())
		ctx.Status(http.StatusUnauthorized)
		return
	}

	switch event.Event {
	case model.PaystackSubscriptionCreate:
		user.PaystackEmailToken = &event.Data.EmailToken
		user.PaystackSubCode = &event.Data.SubCode
		user.PaystackCusCode = &event.Data.Customer.CusCode

		user.Last4 = &event.Data.Authorization.Last4
		user.CardType = &event.Data.Authorization.Brand
		user.ExpMonth = &event.Data.Authorization.ExpMonth
		user.ExpYear = &event.Data.Authorization.ExpYear

		if err := t.DB.SaveUser(&user); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}

		ctx.Status(http.StatusOK)
		return
	case model.PaystackChargeSuccess:
		transc := model.Transactions{
			UserID:      user.ID,
			PaystackRef: event.Data.Reference,
			Amount:      event.Data.Amount,
			Status:      event.Data.Status,
			Channel:     event.Data.Channel,
			Curency:     event.Data.Currency,
			PaidAt:      &event.Data.PaidAt,
		}

		sub := model.Subscriptions{
			UserID:    user.ID,
			Status:    model.StatusActive,
			PlanName:  event.Data.Plan.Name,
			StartDate: time.Now(),
			EndDate:   time.Now().Add(time.Hour * 24 * 30), // TODO: Make a call to figure out the actual expiring date for that subscription
		}

		user.NextPaymentDate = &sub.EndDate

		if err := t.DB.AddTranscSub(&transc, &sub, &user); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}

		ctx.Status(http.StatusOK)
		return
	case model.PaystackInvoiceUpdate:
		if event.Data.Paid == 0 {
			sub := model.Subscriptions{
				UserID:    user.ID,
				Status:    model.StatusFailed,
				PlanName:  event.Data.Plan.Name,
				StartDate: time.Now(),
				EndDate:   event.Data.Subscription.NextPaymentDate,
			}

			grace := user.NextPaymentDate.Add(time.Hour * 24 * 7)
			user.NextPaymentDate = &grace

			if err := t.DB.AddFailedSub(&sub, &user); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}

			amount := fmt.Sprintf("%d", event.Data.Amount)

			t.Email.QueueTransactionFailedEmail(user.UserName, amount, event.Data.Currency, event.Data.Plan.Name, "", user.Email)
		}

		ctx.Status(http.StatusOK)
		return
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

func (t *TranscService) RetryFailedPayment(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized"})
		return
	}

	subID := ctx.Query("id")
	if !model.IsValidUUID(subID) {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid sub id."})
		return
	}

	var sub model.Subscriptions
	if err := t.DB.FetchUserPreloadFailedSub(&sub, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}
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

	body, _ = io.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &sub); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to Unmarshal payload from paystack, err -> ", err.Error())
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse response from paystack"})
		return
	}

	if !sub.Status {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": sub.Message})
		return
	}

	t.Email.QueueSubscriptionCancelled(user.UserName, user.NextPaymentDate.Format("January 02, 2006"), user.Email)

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

	if *user.SubStatus != model.StatusAttention {
		ctx.JSON(http.StatusForbidden, gin.H{"msg": "Can't enable this subscription."})
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
	if err != nil {
		log.Println("[TRANSACTION] An error occured while trying to enable a subscription, err -> ", err, resp.StatusCode)
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to communicate with paystack."})
		return
	}

	log.Println(resp.StatusCode)
	defer resp.Body.Close()
	var sub schema.PaystackSubResp

	body, _ = io.ReadAll(resp.Body)
	log.Println(string(body))

	if err := json.Unmarshal(body, &sub); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to Unmarshal payload from paystack, err -> ", err.Error())
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse response from paystack"})
		return
	}

	if !sub.Status {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": sub.Message})
		return
	}

	t.Email.QueueSubscriptionReEnabled(user.UserName, user.NextPaymentDate.Format("January 02, 2006"), user.Email)

	ctx.JSON(http.StatusOK, gin.H{"msg": sub.Message})
}

// ViewPlans godoc
// @Summary An endpoint to view available plans
// @Description An endpoint to view available plans
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.DocViewPlansResponse "Success"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload"
// @Failure 502 {object} schema.DocNormalResponse "Bad Gateway"
// @Router /api/transc/view/plans [get]
func (t *TranscService) ViewPlans(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
	}

	if plan, err := t.RDB.RetrieveCachedPlans(); err == nil && plan != nil && len(plan) != 0 {
		ctx.JSON(http.StatusOK, gin.H{"plans": plan})
		return
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.paystack.co/plan", nil)
	req.Header.Set("Authorization", "Bearer "+t.SecretKey)

	resp, err := t.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to communicate with paystack."})
		return
	}

	defer resp.Body.Close()
	var plans schema.PaystackViewPlans

	body, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &plans); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to Unmarshal payload from paystack, err -> ", err.Error())
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"msg": "Failed to parse response from paystack."})
		return
	}

	if !plans.Status {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": plans.Message})
		return
	}

	var res []schema.ViewPlansResp

	for _, plan := range plans.Data {
		res = append(res, schema.ViewPlansResp{
			ID:       plan.PlanCode,
			Amount:   plan.Amount,
			Name:     plan.Name,
			Interval: plan.Interval,
			Currency: plan.Currency,
		})
	}

	_ = t.RDB.CachePlans(res)

	ctx.JSON(http.StatusOK, gin.H{"plans": res})
}
