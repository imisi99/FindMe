package core

import (
	"fmt"
	"log"
	"time"

	"github.com/go-mail/mail/v2"
)

type EmailS interface {
	SendEmail(to, subject, body string) error
	SendFriendReqEmail(fromUsername, toUsername, message, viewURL string) (string, string)
	SendForgotPassEmail(username, token string) (string, string)
	SendProjectApplicationEmail(fromUsername, toUsername, message, viewURL string) (string, string)
	SendProjectApplicationAccept(fromUsername, toUsername, message, chatURL string) (string, string)
	SendProjectApplicationReject(fromUsername, toUsername, message, reason string) (string, string)
	SendSubscriptionCreateEmail(username, amount, currency, planName, manageURL string) (string, string)
	SendTransactionFailedEmail(username, amount, currency, planName, retryURL string) (string, string)
	SendSubscriptionReEnabledEmail(username, nextBillingDate string) (string, string)
	SendSubscriptionCancelledEmail(username, endDate string) (string, string)
	SendNotifyFreeTrialEnding(username, endDate, subURL string) (string, string)
}

type Email interface {
	Worker()
	QueueFriendReqEmail(fromUsername, toUsername, message, viewURL, to string)
	QueueForgotPassEmail(to, username, token string)
	QueueProjectApplication(fromUsername, toUsername, message, viewURL, to string)
	QueueProjectApplicationAccept(fromUsername, toUsername, message, chatURL, to string)
	QueueProjectApplicationReject(fromUsername, toUsername, message, reason, to string)
	QueueSubscriptionCreate(username, amount, currency, planName, manageURL, to string)
	QueueTransactionFailedEmail(username, amount, currency, planName, retryURL, to string)
	QueueSubscriptionReEnabled(username, nextBillingDate, to string)
	QueueSubscriptionCancelled(username, endDate, to string)
	QueueNotifyFreeTrialEnding(username, endDate, subURL, to string)
}

type EmailService struct {
	Server   string
	MailPort int
	Addr     string
	Password string
}

type EmailJob struct {
	To          string
	Subject     string
	Body        string
	Attempts    int
	MaxAttempts int
}

type EmailHub struct {
	Jobs       chan *EmailJob
	Quit       chan bool
	WorkerPool int
	Service    EmailS
}

func NewEmailService(server, addr, pass string, port int) *EmailService {
	return &EmailService{Server: server, MailPort: port, Addr: addr, Password: pass}
}

func NewEmailHub(queueSize, workers int, service EmailS) *EmailHub {
	return &EmailHub{
		Jobs:       make(chan *EmailJob, queueSize),
		Quit:       make(chan bool),
		WorkerPool: workers,
		Service:    service,
	}
}

func (h *EmailHub) Run() {
	for range h.WorkerPool {
		go h.Worker()
	}
	log.Println("[EMAIL HUB] The Email hub is up and running")
}

func (h *EmailHub) Stop() {
	for range h.WorkerPool {
		h.Quit <- true
	}
}

func (h *EmailHub) Worker() {
	for {
		select {
		case job := <-h.Jobs:
			err := h.Service.SendEmail(job.To, job.Subject, job.Body)
			if err != nil {
				job.Attempts++
				if job.Attempts <= job.MaxAttempts {
					waitTime := time.Duration(job.Attempts*4) * time.Second
					log.Printf("[EmailJob] Retrying in %v", waitTime)

					go func(j *EmailJob, delay time.Duration) {
						time.Sleep(delay)
						h.Jobs <- j
					}(job, waitTime)
				}
			}
		case <-h.Quit:
			return
		}
	}
}

func (h *EmailHub) QueueFriendReqEmail(fromUsername, toUsername, message, viewURL, to string) {
	body, subject := h.Service.SendFriendReqEmail(fromUsername, toUsername, message, viewURL)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueForgotPassEmail(to, username, token string) {
	body, subject := h.Service.SendForgotPassEmail(username, token)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 3,
	}
}

func (h *EmailHub) QueueProjectApplication(fromUsername, toUsername, message, viewURL, to string) {
	body, subject := h.Service.SendProjectApplicationEmail(fromUsername, toUsername, message, viewURL)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueProjectApplicationAccept(fromUsername, toUsername, message, chatURL, to string) {
	body, subject := h.Service.SendProjectApplicationAccept(fromUsername, toUsername, message, chatURL)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueProjectApplicationReject(fromUsername, toUsername, message, reason, to string) {
	body, subject := h.Service.SendProjectApplicationReject(fromUsername, toUsername, message, reason)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueSubscriptionCreate(username, amount, currency, planName, manageURL, to string) {
	body, subject := h.Service.SendSubscriptionCreateEmail(username, amount, currency, planName, manageURL)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueTransactionFailedEmail(username, amount, currency, planName, retryURL, to string) {
	body, subject := h.Service.SendTransactionFailedEmail(username, amount, currency, planName, retryURL)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueSubscriptionReEnabled(username, nextBillingDate, to string) {
	body, subject := h.Service.SendSubscriptionReEnabledEmail(username, nextBillingDate)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueSubscriptionCancelled(username, endDate, to string) {
	body, subject := h.Service.SendSubscriptionCancelledEmail(username, endDate)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

func (h *EmailHub) QueueNotifyFreeTrialEnding(username, endDate, subURL, to string) {
	body, subject := h.Service.SendNotifyFreeTrialEnding(username, endDate, subURL)
	h.Jobs <- &EmailJob{
		To:          to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 2,
	}
}

// SendForgotPassEmail -> Sends an OTP for reseting Password
func (e *EmailService) SendForgotPassEmail(username, token string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Password Reset</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">

	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#4f46e5; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
				<h1 style="margin:0; font-size:22px; color:#ffffff;">Reset Your Password</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
				<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
				<p style="font-size:15px; color:#374151; margin-bottom:20px;">
					We received a request to reset your password. Please use the OTP code below:
				</p>
				<div style="text-align:center; margin:30px 0;">
					<span style="display:inline-block; background:#4f46e5; color:#ffffff; font-size:24px; font-weight:bold; padding:12px 24px; border-radius:6px;">
					%s
					</span>
				</div>
				<p style="font-size:14px; color:#6b7280; margin-bottom:10px;">
					This OTP is valid for the next 10 minutes.
				</p>
				<p style="font-size:14px; color:#6b7280;">
					If you didn’t request this, you can safely ignore this email.
				</p>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
				This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>

	</body>
	</html>`, username, token)

	return htmlBody, "Password reset OTP"
}

// SendFriendReqEmail -> Sends a notification about a new friend request
func (e *EmailService) SendFriendReqEmail(fromUsername, toUsername, message, viewURL string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>New Friend Request</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">

	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#4f46e5; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">New Friend Request</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						<b>%s</b> has sent you a friend request!
					</p>
					<p style="font-size:14px; color:#374151; margin:20px 0;">
						<span style="font-weight:bold;">Message:</span>
					</p>
					<blockquote style="margin:0; padding:15px; background:#f3f4f6; border-left:4px solid #187e5fff; font-style:italic; font-size:14px; color:#1f2937;">
						%s
					</blockquote>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						You can view the request below:
					</p>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="%s" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold; margin-right:10px;">View Request</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					You are receiving this email because you have an account on <b>FindMe</b>.<br/>
					If you did not expect this, you can safely ignore this email.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>

	</body>
	</html>`,
		toUsername,
		fromUsername,
		message,
		viewURL,
	)

	return htmlBody, "New Friend Request"
}

// SendProjectApplicationEmail -> Sends a notification about a new application to post
func (e *EmailService) SendProjectApplicationEmail(fromUsername, toUsername, message, viewURL string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>New Project Application Request</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">

	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#4f46e5; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">New Project Application Request</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						<b>%s</b> has applied for a post created by you.
					</p>
					<p style="font-size:14px; color:#374151; margin:20px 0;">
						<span style="font-weight:bold;">Project Description:</span>
					</p>
					<blockquote style="margin:0; padding:15px; background:#f3f4f6; border-left:4px solid #758f19ff; font-style:italic; font-size:14px; color:#1f2937;">
						%s
					</blockquote>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						You can view the application below:
					</p>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="%s" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold; margin-right:10px;">View Application</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					You are receiving this email because you have an account on <b>FindMe</b>.<br/>
					If you did not expect this, you can safely ignore this email.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>

	</body>
	</html>`,
		toUsername,
		fromUsername,
		message,
		viewURL,
	)

	return htmlBody, "New Project Application Request"
}

// SendProjectApplicationAccept -> Sends notification about accepted post application
func (e *EmailService) SendProjectApplicationAccept(fromUsername, toUsername, message, chatURL string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Application Update</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">

	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#4f46e5; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">Application Update</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						<b>%s</b> has accepted your application!
					</p>
					<p style="font-size:14px; color:#374151; margin:20px 0;">
						<span style="font-weight:bold;">You can now work together on the project with description:</span>
					</p>
					<blockquote style="margin:0; padding:15px; background:#f3f4f6; border-left:4px solid #26868aff; font-style:italic; font-size:14px; color:#1f2937;">
						%s
					</blockquote>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						You can chat with each other:
					</p>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="%s" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold; margin-right:10px;">Chat</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					You are receiving this email because you have an account on <b>FindMe</b>.<br/>
					If you did not expect this, you can safely ignore this email.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>

	</body>
	</html>`,
		toUsername,
		fromUsername,
		message,
		chatURL,
	)

	return htmlBody, "Project Application Update"
}

// SendProjectApplicationReject -> Sends notification about rejected post application
func (e *EmailService) SendProjectApplicationReject(fromUsername, toUsername, message, reason string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Application Update</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">

	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#4f46e5; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">Application Update</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						<b>%s</b> has rejected your application!
					</p>
					<p style="font-size:14px; color:#374151; margin:20px 0;">
						<span style="font-weight:bold;">you've been rejected from project with description</span>
					</p>
					<blockquote style="margin:0; padding:15px; background:#f3f4f6; border-left:4px solid #26868aff; font-style:italic; font-size:14px; color:#1f2937;">
						%s
					</blockquote>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						You weren't accepted due to:
					</p>
					<blockquote style="margin:0; padding:15px; background:#f3f4f6; border-left:4px solid #26868aff; font-style:italic; font-size:14px; color:#1f2937;">
						%s
					</blockquote>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					You are receiving this email because you have an account on <b>FindMe</b>.<br/>
					If you did not expect this, you can safely ignore this email.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>

	</body>
	</html>`,
		toUsername,
		fromUsername,
		message,
		reason,
	)

	return htmlBody, "Project Application Update"
}

// SendSubscriptionCreateEmail -> Sends notification about a subscription creation
func (e *EmailService) SendSubscriptionCreateEmail(username, amount, currency, planName, manageURL string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Subscription created</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#dc2626; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">Subscription created</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						Your subscription has been successfully created and you now have access to our premium features.
						We are excited to have you on board
					</p>
					<table width="100%%" cellpadding="10" cellspacing="0" border="0" style="background:#fef2f2; border-radius:6px; border:1px solid #fecaca; margin-bottom:20px;">
						<tr>
							<td style="font-size:14px; color:#6b7280; border-bottom:1px solid #fecaca;">Plan</td>
							<td style="font-size:14px; color:#111827; font-weight:bold; border-bottom:1px solid #fecaca; text-align:right;">%s</td>
						</tr>
						<tr>
							<td style="font-size:14px; color:#6b7280;">Amount</td>
							<td style="font-size:14px; color:#111827; font-weight:bold; text-align:right;">%s %s</td>
						</tr>
					</table>
					<p style="font-size:14px; color:#374151; margin-bottom:10px;">
						<b>What happens next?</b>
					</p>
					<ul style="font-size:14px; color:#6b7280; margin-bottom:20px; padding-left:20px;">
						<li style="margin-bottom:8px;">You now have access to all our premium features</li>
						<li style="margin-bottom:8px;">You will have a recurring payment for this subscription and can cancel at anytime</li>
					</ul>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="%s" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold;">Manage Subscription</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					You are receiving this email because you have a subscription on <b>FindMe</b>.<br/>
					If you believe this is an error, please contact our support team.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>
	</body>
	</html>`,
		username,
		planName,
		currency,
		amount,
		manageURL,
	)
	return htmlBody, "Subscription Created - FindMe"
}

// SendTransactionFailedEmail -> Sends notification about a failed transaction for a subscription
func (e *EmailService) SendTransactionFailedEmail(username, amount, currency, planName, retryURL string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Payment Failed</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#dc2626; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">Payment Failed</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						We were unable to process your subscription payment. This could be due to insufficient funds, an expired card, or a temporary issue with your bank.
					</p>
					<table width="100%%" cellpadding="10" cellspacing="0" border="0" style="background:#fef2f2; border-radius:6px; border:1px solid #fecaca; margin-bottom:20px;">
						<tr>
							<td style="font-size:14px; color:#6b7280; border-bottom:1px solid #fecaca;">Plan</td>
							<td style="font-size:14px; color:#111827; font-weight:bold; border-bottom:1px solid #fecaca; text-align:right;">%s</td>
						</tr>
						<tr>
							<td style="font-size:14px; color:#6b7280;">Amount</td>
							<td style="font-size:14px; color:#111827; font-weight:bold; text-align:right;">%s %s</td>
						</tr>
					</table>
					<p style="font-size:14px; color:#374151; margin-bottom:10px;">
						<b>What happens next?</b>
					</p>
					<ul style="font-size:14px; color:#6b7280; margin-bottom:20px; padding-left:20px;">
						<li style="margin-bottom:8px;">You have a <b>7-day grace period</b> to update your payment method</li>
						<li style="margin-bottom:8px;">Your subscription will remain active during this period</li>
						<li>After the grace period, your subscription will be paused</li>
					</ul>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="%s" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold;">Manage your subscription</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					You are receiving this email because you have a subscription on <b>FindMe</b>.<br/>
					If you believe this is an error, please contact our support team.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>
	</body>
	</html>`,
		username,
		planName,
		currency,
		amount,
		retryURL,
	)
	return htmlBody, "Action Required: Payment Failed - FindMe"
}

// SendSubscriptionReEnabledEmail -> Sends a notification about a re-enabled subscription
func (e *EmailService) SendSubscriptionReEnabledEmail(username, nextBillingDate string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Subscription Re-enabled</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#059669; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">Welcome Back!</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						Great news! Your subscription has been re-enabled. You'll continue to enjoy uninterrupted access to all premium features.
					</p>
					<table width="100%%" cellpadding="10" cellspacing="0" border="0" style="background:#ecfdf5; border-radius:6px; border:1px solid #a7f3d0; margin-bottom:20px;">
						<tr>
							<td style="font-size:14px; color:#6b7280;">Status</td>
							<td style="font-size:14px; color:#059669; font-weight:bold; text-align:right;">Active</td>
						</tr>
						<tr>
							<td style="font-size:14px; color:#6b7280; border-top:1px solid #a7f3d0;">Next Billing Date</td>
							<td style="font-size:14px; color:#111827; font-weight:bold; text-align:right; border-top:1px solid #a7f3d0;">%s</td>
						</tr>
					</table>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						Your card on file will be charged automatically on the next billing date.
					</p>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="https://findme.app/dashboard" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold;">Go to Dashboard</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					Thank you for staying with us!<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>
	</body>
	</html>`,
		username,
		nextBillingDate,
	)
	return htmlBody, "Your FindMe Subscription Has Been Re-enabled"
}

// SendSubscriptionCancelledEmail -> Sends a notification for a cancelled subscription
func (e *EmailService) SendSubscriptionCancelledEmail(username, endDate string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Subscription Cancelled</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#6b7280; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">Subscription Cancelled</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						Your subscription has been cancelled. You will <b>not</b> be billed on your next payment date.
					</p>
					<table width="100%%" cellpadding="10" cellspacing="0" border="0" style="background:#f3f4f6; border-radius:6px; margin-bottom:20px;">
						<tr>
							<td style="font-size:14px; color:#6b7280;">Access Until</td>
							<td style="font-size:14px; color:#111827; font-weight:bold; text-align:right;">%s</td>
						</tr>
					</table>
					<p style="font-size:14px; color:#374151; margin-bottom:20px;">
						You'll continue to have full access to all premium features until this date. After that, your account will revert to the free plan.
					</p>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						Changed your mind? You can re-enable your subscription anytime before it expires.
					</p>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="https://findme.app/settings/subscription" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold;">Manage Subscription</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					We're sorry to see you go. If you have any feedback, we'd love to hear it.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>
	</body>
	</html>`,
		username,
		endDate,
	)
	return htmlBody, "Your FindMe Subscription Has Been Cancelled"
}

// SendNotifyFreeTrialEnding -> Sends an notification for a free trial ending
func (e *EmailService) SendNotifyFreeTrialEnding(username, endDate, subURL string) (string, string) {
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>Your Free Trial is Ending</title>
	</head>
	<body style="margin:0; padding:0; background:#f9fafb; font-family:Arial, sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f9fafb; padding:40px 0;">
		<tr>
		<td align="center">
			<table width="600" cellpadding="0" cellspacing="0" border="0" style="background:#ffffff; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.05);">
			<tr>
				<td style="background:#f59e0b; padding:20px; text-align:center; border-top-left-radius:8px; border-top-right-radius:8px;">
					<h1 style="margin:0; font-size:22px; color:#ffffff;">Your Free Trial is Ending Soon</h1>
				</td>
			</tr>
			<tr>
				<td style="padding:30px;">
					<p style="font-size:16px; color:#111827; margin-bottom:20px;">Hello %s,</p>
					<p style="font-size:15px; color:#374151; margin-bottom:20px;">
						Your free trial ends on <b>%s</b>. We hope you've been enjoying FindMe!
					</p>
					<p style="font-size:14px; color:#374151; margin-bottom:10px;">
						<b>Here's what you'll lose access to:</b>
					</p>
					<ul style="font-size:14px; color:#6b7280; margin-bottom:20px; padding-left:20px;">
						<li style="margin-bottom:8px;">Advanced skill matching algorithms to recommend projects suited to you</li>
					</ul>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						Subscribe now to keep your premium access without any interruption.
					</p>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="%s" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold;">Subscribe Now</a>
					</div>
				</td>
			</tr>
			<tr>
				<td style="padding:20px; text-align:center; font-size:12px; color:#9ca3af;">
					If you have any questions, feel free to reach out to our support team.<br/><br/>
					This is an automated email, please do not reply.
				</td>
			</tr>
			</table>
		</td>
		</tr>
	</table>
	</body>
	</html>`,
		username,
		endDate,
		subURL,
	)
	return htmlBody, "Your FindMe Free Trial is Ending Soon"
}

func (e *EmailService) SendEmail(to, subject, body string) error {
	msg := mail.NewMessage()
	msg.SetAddressHeader("From", e.Addr, "FindMe Team")
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	mail := mail.NewDialer(e.Server, e.MailPort, e.Addr, e.Password)

	if err := mail.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}
