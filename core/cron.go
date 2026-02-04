package core

import (
	"log"
	"time"

	"findme/model"

	"github.com/robfig/cron/v3"
)

type CronWorker interface {
	TrialEndingReminders() error
}

type Cron struct {
	DB    DB
	Email Email
	Cron  *cron.Cron
}

func NewCron(db DB, email Email, cron *cron.Cron) *Cron {
	return &Cron{db, email, cron}
}

func (c *Cron) TrialEndingReminders() error {
	_, err := c.Cron.AddFunc("0 9 * * *", func() {
		log.Println("[CRON] Running trial ending reminders...")
		twoDays := time.Now().Add(time.Hour * 24 * 2)
		today := time.Now()

		var users []model.User
		if err := c.DB.FetchTrialEndingUsers(&users, twoDays, today); err != nil {
			return
		}

		if len(users) == 0 {
			return
		}

		ids := make([]string, len(users))

		for i, user := range users {
			ids[i] = user.ID
			c.Email.QueueNotifyFreeTrialEnding(user.UserName, user.FreeTrial.Format("January 2, 2006"), "", user.Email)
		}

		if err := c.DB.UpdateSentReminder(ids); err != nil {
			return
		}
	})

	return err
}
