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

type Transc interface {
	GetTransactions(ctx *gin.Context)
	InitializeTransaction(ctx *gin.Context)
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
// @Description An endpoint for initializing a transaction on paystack
// @Tags Transaction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param amount query string true "amount"
// @Success 200 {object} schema.DocInitTranscResponse "Success"
// @Failure 400 {object} schema.DocNormalResponse "Bad Query"
// @Failure 401 {object} schema.DocNormalResponse "Unauthorized"
// @Failure 404 {object} schema.DocNormalResponse "Record not found"
// @Failure 500 {object} schema.DocNormalResponse "Server error"
// @Failure 502 {object} schema.DocNormalResponse "Bad Gateway"
// @Router /api/transc/initialize [post]
func (t *TranscService) InitializeTransaction(ctx *gin.Context) {
	uid, tp := ctx.GetString("userID"), ctx.GetString("purpose")
	if !model.IsValidUUID(uid) || tp != "login" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "Unauthorized user."})
		return
	}

	amount := ctx.Query("amount")
	if amount == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"msg": "Transaction amount not in query."})
		return
	}

	var user model.User
	if err := t.DB.FetchUser(&user, uid); err != nil {
		cm := err.(*core.CustomMessage)
		ctx.JSON(cm.Code, gin.H{"msg": cm.Message})
		return
	}

	payload := map[string]string{"email": user.Email, "amount": amount}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Println("[TRANSACTION] An error occured while trying to encode body to initialize paystack transaction, err -> ", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to parse payload."})
		return
	}

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
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to parse paystack response"})
		return
	}

	if !paystack.Status || paystack.Data.AuthorizationURL == "" || paystack.Data.AccessCode == "" || paystack.Data.Reference == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{"msg": "An error occured while interacting with paystack api."})
		return } ctx.JSON(http.StatusOK, gin.H{"uri": paystack.Data.AuthorizationURL, "token": paystack.Data.AccessCode}) }

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
		ctx.Status(http.StatusBadRequest)
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
					StartDate: time.Now(),
					EndDate:   time.Now().Add(time.Hour * 24 * 30),
				}

				if err := t.DB.AddTranscSub(&transc, &sub); err != nil {
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
				StartDate:     time.Now(),
				EndDate:       time.Now().Add(time.Hour * 24 * 30),
			}

			if err := t.DB.SaveTranscAddSub(&transc, &sub); err != nil {
				ctx.Status(http.StatusInternalServerError)
				return
			}
		}
		ctx.Status(http.StatusOK)
	case model.InvoiceUpdate:
		if !event.Data.Paid {
				t.Email.QueueTransactionFailedEmail()
		}
		t.Email.QueueTransactionSuccessEmail()
	default:
		return
	}
}
