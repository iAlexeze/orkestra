package notification

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/orkspace/orkestra/pkg/types"
)

func sendEmailNotification(
	_ context.Context,
	host string,
	port int,
	user, pass, from string,
	to []string,
	operatorName, teamName string,
	cond types.Condition,
	data map[string]interface{},
) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	auth := smtp.PlainAuth("", user, pass, host)

	subject := fmt.Sprintf("[Orkestra] Condition triggered: %s (%s)", cond.Field, operatorName)
	body := fmt.Sprintf(
		"Team: %s\nOperator: %s\nField: %s\nNotify: %v\n\nCondition has evaluated to true.\n",
		teamName, operatorName, cond.Field, cond.Notify,
	)

	msg := []byte("To: " + to[0] + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	return smtp.SendMail(addr, auth, from, to, msg)
}
