package monitor

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// msg is a RFC 822-style email with headers first, a blank line, and then the message body.
// The lines of msg should be CRLF terminated.
func sendmail(from string, to []string, msg []byte) error {
	mx, err := lookupMX(from)
	if err != nil {
		return fmt.Errorf("failed to lookup MX for %v: %v", from, err)
	}

	err = nil
	for _, v := range mx {
		err = smtp.SendMail(fmt.Sprintf("%v:25", v.Host), nil, from, to, msg)
		if err != nil {
			continue
		}
		// Sent
		return nil
	}

	return err
}

func lookupMX(email string) ([]*net.MX, error) {
	tokens := strings.Split(email, "@")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("invalid email address: %v", email)
	}

	return net.LookupMX(tokens[1])
}
