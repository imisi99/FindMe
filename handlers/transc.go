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

// TODO:
// Initiate Subscription
// Handle Automatic Subscription
// Handle Failed Subscription
// Cancel Subscription
// Enable Subscription
// Add a Sub Create email for notifying new sub users
// Maintain a correct card info on user updating card

type Transc interface {
	GetTransactions(ctx *gin.Context)
	InitializeTransaction(ctx *gin.Context)
	UpdateSubscriptionCard(ctx *gin.Context)
	CancelSubscription(ctx *gin.Context)
	EnableSubscription(ctx *gin.Context)
	ViewPlans(ctx *gin.Context)
	VerifyTranscWebhook(ctx *gin.Context)
	RetryFailedPayment(ctx *gin.Context)
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
// @Tags Transaction
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
		user.PaystackAuthCode = &event.Data.Authorization.AuthCode

		user.Last4 = &event.Data.Authorization.Last4
		user.CardType = &event.Data.Authorization.Brand
		user.ExpMonth = &event.Data.Authorization.ExpMonth
		user.ExpYear = &event.Data.Authorization.ExpYear

		if err := t.DB.SaveUser(&user); err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}

		t.Email.QueueSubscriptionCreate(user.UserName, "", event.Data.Currency, event.Data.Plan.Name, "", user.Email)

		ctx.Status(http.StatusOK)
		return
	case model.PaystackChargeSuccess:

		// Handling charges for updating card and stuff
		if event.Data.Amount == 5000 {
			user.PaystackAuthCode = &event.Data.Authorization.AuthCode

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
		}

		transc := model.Transactions{
			UserID:      user.ID,
			PaystackRef: event.Data.Reference,
			Amount:      event.Data.Amount,
			Status:      event.Data.Status,
			Channel:     event.Data.Channel,
			Curency:     event.Data.Currency,
			PaidAt:      &event.Data.PaidAt,
		}

		isManual := false
		sid := ""

		if event.Data.Metadata != nil {
			if metadata, ok := event.Data.Metadata.(map[string]any); ok {
				if chargeType, exists := metadata["charge_type"]; exists && chargeType == "manual_retry" {
					isManual = true
					if subID, ok := metadata["sub_id"]; ok {
						sid = subID.(string)
					}
				}
			}
		}

		if isManual && model.IsValidUUID(sid) {
			var sub model.Subscriptions
			if err := t.DB.FetchSub(&sub, sid); err != nil {
				log.Println("[TRANSACTION] An error occured while trying to fetch sub for manual retry payment, err -> ", err.Error())
				ctx.Status(http.StatusInternalServerError)
				return
			}

			sub.Status = model.StatusActive
			user.LastSub = &sub.EndDate
			user.NextPaymentDate = user.LastSub

			if err := t.DB.AddTranscSaveSub(&transc, &sub, &user); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}
		} else {
			details, err := t.FetchSubscriptionDetails(*user.PaystackSubCode)
			if err != nil || details == nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}

			sub := model.Subscriptions{
				UserID:    user.ID,
				Status:    model.StatusActive,
				PlanName:  event.Data.Plan.Name,
				StartDate: time.Now(),
				EndDate:   details.Data.NextPaymentDate,
			}

			user.NextPaymentDate = &sub.EndDate
			user.LastSub = user.NextPaymentDate

			if err := t.DB.AddTranscSub(&transc, &sub, &user); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}
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

			grace := user.LastSub.Add(time.Hour * 24 * 7)
			user.LastSub = &grace

			if err := t.DB.AddFailedSub(&sub, &user); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}

			amount := fmt.Sprintf("%d", event.Data.Amount)
			t.Email.QueueTransactionFailedEmail(user.UserName, amount, event.Data.Currency, event.Data.Plan.Name, "", user.Email)
		}

		ctx.Status(http.StatusOK)
		return
	case model.PaystackSubscriptionNotRenew:
		t.Email.QueueSubscriptionCancelled(user.UserName, user.NextPaymentDate.Format("January 02, 2006"), user.Email)

		user.NextPaymentDate = nil
		if err := t.DB.SaveUser(&user); err != nil {
			ctx.Status(http.StatusInternalServerError)
		}
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

// RetryFailedPayment godoc
// @Summary Retries a failed payment for subscription
// @Description An endpoint for retrying a failed subscription payment
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "Sub ID"
// @Success 200 {object} schema.DocNormalResponse "Success"
// @Failure 400 {object} schema.DocNormalResponse "Bad Request"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 402 {object} schema.DocMsgResResponse "Payment required"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 422 {object} schema.DocNormalResponse "Failed to parse payload / response"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "External server error"
// @Router /api/transc/retry-payment [post]
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

	var user model.User
	if err := t.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var sub model.Subscriptions
	if err := t.DB.FetchSub(&sub, subID); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	if sub.Status != model.StatusFailed {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Subscription is not failed and can't be retried."})
		return
	}

	plans, err := t.RetrievePlans()
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	var subPlan schema.ViewPlansResp
	for _, plan := range plans {
		if sub.PlanName == plan.Name {
			subPlan = plan
			break
		}
	}

	amount := fmt.Sprintf("%d", subPlan.Amount)
	chargePayload := map[string]any{
		"email":              user.Email,
		"amount":             amount,
		"authorization_code": *user.PaystackAuthCode,
		"metadata": map[string]any{
			"charge_type": "manual_retry",
			"sub_id":      subID,
		},
	}

	chargeBody, _ := json.Marshal(chargePayload)

	chargeReq, _ := http.NewRequest(http.MethodPost, "https://api.paystack.co/transaction/charge_authorization", bytes.NewBuffer(chargeBody))
	chargeReq.Header.Set("Authorization", "Bearer "+t.SecretKey)
	chargeReq.Header.Set("Content-Type", "application/json")

	chargeResp, err := t.Client.Do(chargeReq)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Failed to communicate with paystack to make payment."})
		return
	}

	defer chargeResp.Body.Close()

	var chargeResponse schema.AuthCharge
	chargeData, _ := io.ReadAll(chargeResp.Body)

	if err := json.Unmarshal(chargeData, &chargeResponse); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to unmarshal the response of paystack, err -> ", err.Error())
		ctx.JSON(http.StatusBadGateway, gin.H{"msg": "Unable to communicate with paystack to initiate transaction."})
		return
	}

	if !chargeResponse.Status || chargeResponse.Data.Status != model.StatusSuccess {
		ctx.JSON(http.StatusPaymentRequired, gin.H{"msg": "Failed payment!", "reason": chargeResponse.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"msg": "Payment successful your subscription will be renewed."})
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
		return
	}

	plans, err := t.RetrievePlans()
	if err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"plans": plans})
}

// RetrievePlans -> A helper func to retrieve plans from cache if available or fetch from paystack
func (t *TranscService) RetrievePlans() ([]schema.ViewPlansResp, error) {
	if plan, err := t.RDB.RetrieveCachedPlans(); err == nil && plan != nil && len(plan) != 0 {
		return plan, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.paystack.co/plan", nil)
	req.Header.Set("Authorization", "Bearer "+t.SecretKey)

	resp, err := t.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, &core.CustomMessage{Code: http.StatusBadGateway, Message: "Failed to communicate with paystack."}
	}

	defer resp.Body.Close()
	var plans schema.PaystackViewPlans

	body, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &plans); err != nil {
		log.Println("[TRANSACTION] An error occured while trying to Unmarshal payload from paystack, err -> ", err.Error())
		return nil, &core.CustomMessage{Code: http.StatusUnprocessableEntity, Message: "Failed to parse response from paystack."}
	}

	if !plans.Status {
		return nil, &core.CustomMessage{Code: http.StatusBadGateway, Message: plans.Message}
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
	return res, nil
}

// FetchSubscriptionDetails -> A helper func to fetch the subscription details from paystack
func (t *TranscService) FetchSubscriptionDetails(subCode string) (*schema.SubscriptionDetails, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.paystack.co/subscription/"+subCode, nil)
	req.Header.Set("Authorization", "Bearer "+t.SecretKey)

	resp, err := t.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, &core.CustomMessage{Code: http.StatusBadGateway, Message: "Failed to communicate with paystack."}
	}

	defer resp.Body.Close()

	var sub schema.SubscriptionDetails
	body, _ := io.ReadAll(resp.Body)

	if err := json.Unmarshal(body, &sub); err != nil {
		log.Println("[TRANSACTION] Failed to unmarshal paystack response for subscription details, err -> ", err.Error())
		return nil, &core.CustomMessage{Code: http.StatusUnprocessableEntity, Message: "Failed to unmarshal paystack response"}
	}

	return &sub, nil
}
