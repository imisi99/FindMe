package core

import (
	"fmt"
	"log"
	"time"

	"github.com/go-mail/mail/v2"
)

type Email interface {
	SendForgotPassEmail(username, token string) (string, string)
	SendFriendReqEmail(fromUsername, toUsername, message, viewURL string) (string, string)
	SendProjectApplicationEmail(fromUsername, toUsername, message, viewURL string) (string, string)
	SendProjectApplicationAccept(fromUsername, toUsername, message, chatURL string) (string, string)
	SendProjectApplicationReject(fromusername, toUsername, message, reason string) (string, string)
	SendEmail(to, subject, body string) error
}

type MyEmail struct {
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
}

func NewEmail(server, addr, pass string, port int) *MyEmail {
	return &MyEmail{Server: server, MailPort: port, Addr: addr, Password: pass}
}

func NewEmailHub(queueSize, workers int) *EmailHub {
	return &EmailHub{
		Jobs:       make(chan *EmailJob, queueSize),
		Quit:       make(chan bool),
		WorkerPool: workers,
	}
}

func (h *EmailHub) Run(emailService Email) {
	for range h.WorkerPool {
		go h.Worker(emailService)
	}
	log.Println("[EMAIL HUB] The Email hub is up and running")
}

func (h *EmailHub) Stop() {
	for range h.WorkerPool {
		h.Quit <- true
	}
}

func (h *EmailHub) Worker(emailService Email) {
	for {
		select {
		case job := <-h.Jobs:
			err := emailService.SendEmail(job.To, job.Subject, job.Body)
			if err != nil {
				job.Attempts++
				if job.Attempts <= job.MaxAttempts {
					waitTime := time.Duration(job.Attempts*3) * time.Second
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

// SendForgotPassEmail -> Sends an OTP for reseting Password
func (e *MyEmail) SendForgotPassEmail(username, token string) (string, string) {
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
func (e *MyEmail) SendFriendReqEmail(fromUsername, toUsername, message, viewURL string) (string, string) {
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
func (e *MyEmail) SendProjectApplicationEmail(fromUsername, toUsername, message, viewURL string) (string, string) {
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
func (e *MyEmail) SendProjectApplicationAccept(fromUsername, toUsername, message, chatURL string) (string, string) {
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
func (e *MyEmail) SendProjectApplicationReject(fromUsername, toUsername, message, reason string) (string, string) {
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

func (e *MyEmail) SendEmail(to, subject, body string) error {
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
