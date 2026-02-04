package schema

import (
	"time"
)

type TransactionResponse struct {
	ID      string
	Amount  int64
	Channel string
	Status  string
	PaidAt  time.Time
}

type InitTransaction struct {
	Status bool `json:"status" binding:"required"`
	Data   struct {
		AuthorizationURL string `json:"authorization_url" binding:"omitempty"`
		AccessCode       string `json:"access_code" binding:"omitempty"`
		Reference        string `json:"reference" binding:"omitempty"`
	} `json:"data" binding:"required"`
}

type PaystackEvent struct {
	Event string `json:"event" binding:"required"`
	Data  struct {
		Status        string    `json:"status" binding:"omitempty"`
		Reference     string    `json:"reference" binding:"omitempty"`
		Channel       string    `json:"channel" binding:"omitempty"`
		Currency      string    `json:"currency" binding:"omitempty"`
		Amount        string    `json:"amount" binding:"omitempty"`
		SubCode       string    `json:"subscription_code" binding:"omitempty"`
		PaidAt        time.Time `json:"paid_at" binding:"omitempty"`
		Paid          bool      `json:"paid" binding:"omitempty"`
		Authorization struct {
			Last4    string `json:"last_4" binding:"omitempty"`
			Brand    string `json:"brand" binding:"omitempty"`
			ExpMonth string `json:"exp_month" binding:"omitempty"`
			ExpYear  string `json:"exp_year" binding:"omitempty"`
		}
		Customer struct {
			CusCode string `json:"customer_code" binding:"omitempty"`
			Email   string `json:"email" binding:"omitempty"`
		}
		Transaction struct {
			Reference string `json:"reference" binding:"omitempty"`
		} `json:"transaction" binding:"omitempty"`
		Subscription struct {
			Status          string    `json:"status" binding:"omitempty"`
			NextPaymentDate time.Time `json:"next_payment_date" binding:"omitempty"`
		}
		Plan struct {
			Name string
		}
	} `json:"data" binding:"required"`
}

type PaystackUpdateCard struct {
	Status  bool   `json:"status" binding:"required"`
	Message string `json:"message" binding:"required"`
	Data    struct {
		Link string `json:"link" binding:"omitempty"`
	} `json:"data" binding:"omitempty"`
}

type PaystackViewSub struct {
	Status bool `json:"status" binding:"required"`
	Data   struct {
		EmailToken string `json:"email_token" binding:"omitempty"`
	} `json:"data" binding:"omitempty"`
}

type PaystackSubResp struct {
	Status  bool   `json:"status" binding:"required"`
	Message string `json:"message" binding:"required"`
}

type PaystackViewPlans struct {
	Status  bool   `json:"status" binding:"required"`
	Message string `json:"message" binding:"required"`
	Data    []struct {
		Name     string `json:"name" binding:"required"`
		Interval string `json:"Interval" binding:"required"`
		PlanCode string `json:"plan_code" binding:"omitempty"`
		Currency string `json:"currency" binding:"required"`
		Amount   int64  `json:"amount" binding:"required"`
	} `json:"data" binding:"omitempty"`
}

type ViewPlansResp struct {
	ID       string
	Name     string
	Amount   int64
	Interval string
	Currency string
}
