package schema

import "time"

type TransactionResponse struct {
	ID      string
	Amount  int64
	Channel string
	Status  string
	PaidAt  time.Time
}
