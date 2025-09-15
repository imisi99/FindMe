package core

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/gomail.v2"
)


var mailPort = 587

func SendForgotPassEmail(email string, username string, token string) error {
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

	if os.Getenv("Testing") == "True" {return nil} // For testing the forgot password endpoint

	msg := gomail.NewMessage()
	msg.SetHeader("From", os.Getenv("EMAIL"))
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "Password reset OTP")
	msg.SetBody("text/html", htmlBody)

	mail := gomail.NewDialer("smtp.gmail.com", mailPort, os.Getenv("EMAIL"), os.Getenv("EMAIL_APP_PASSWORD"))

	if err := mail.DialAndSend(msg); err != nil {
		log.Printf("Unable to send email -> %v", err)
		return err
	}
	return nil
}


func SendFriendReqEmail(email string, fromUsername string, toUsername string, message string, acceptURL string, rejectURL string) error {
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
					<blockquote style="margin:0; padding:15px; background:#f3f4f6; border-left:4px solid #4f46e5; font-style:italic; font-size:14px; color:#1f2937;">
						%s
					</blockquote>
					<p style="font-size:14px; color:#6b7280; margin-bottom:30px;">
						Click one of the options below to respond:
					</p>
					<div style="text-align:center; margin-bottom:30px;">
						<a href="%s" style="background:#4f46e5; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold; margin-right:10px;">Accept</a>
						<a href="%s" style="background:#e11d48; color:#ffffff; padding:12px 24px; text-decoration:none; border-radius:6px; font-size:15px; font-weight:bold;">Reject</a>
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
		acceptURL, 
		rejectURL,
	)

	if os.Getenv("Testing") == "True" {return nil}  // For tests not ideal

	msg := gomail.NewMessage()
	msg.SetHeader("From", os.Getenv("EMAIL"))
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "New Friend Request")
	msg.SetBody("text/html", htmlBody)

	mail := gomail.NewDialer("smtp.gmail.com", mailPort, os.Getenv("EMAIL"), os.Getenv("EMAIL_APP_PASSWORD"))

	if err := mail.DialAndSend(msg); err != nil {
		log.Printf("Unable to send email -> %v", err)
		return err
	}
	return nil
}
