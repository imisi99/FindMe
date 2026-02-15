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
	Event string `json:"event"`
	Data  struct {
		Status        string    `json:"status"`
		Reference     string    `json:"reference"`
		Channel       string    `json:"channel"`
		Currency      string    `json:"currency"`
		Amount        int64     `json:"amount"`
		SubCode       string    `json:"subscription_code"`
		EmailToken    string    `json:"email_token"`
		PaidAt        time.Time `json:"paid_at"`
		Paid          int       `json:"paid"`
		Authorization struct {
			AuthCode string `json:"authorization_code"`
			Last4    string `json:"last4"`
			Brand    string `json:"brand"`
			ExpMonth string `json:"exp_month"`
			ExpYear  string `json:"exp_year"`
		} `json:"authorization"`
		Customer struct {
			CusCode string `json:"customer_code"`
			Email   string `json:"email"`
		} `json:"customer"`
		Transaction struct {
			Reference string `json:"reference"`
		} `json:"transaction"`
		Subscription struct {
			Status          string    `json:"status"`
			SubCode         string    `json:"subscription_code"`
			EmailToken      string    `json:"email_token"`
			NextPaymentDate time.Time `json:"next_payment_date"`
		} `json:"subscription"`
		Plan struct {
			Name string
		} `json:"plan"`
		Metadata any `json:"metadata"`
	} `json:"data"`
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

type PaymentInfo struct {
	Last4           string
	Month           string
	Year            string
	Card            string
	NextPaymentDate time.Time
}

type AuthCharge struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Status string `json:"status"`
	} `json:"data"`
}

type SubscriptionDetails struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Status          string    `json:"status"`
		NextPaymentDate time.Time `json:"next_payment_date"`
	}
}
