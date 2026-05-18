package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"time"
)

type EmailReachabilityError struct {
	Message string
}

func (e *EmailReachabilityError) Error() string {
	return e.Message
}

func IsEmailReachabilityError(err error) bool {
	var target *EmailReachabilityError
	return errors.As(err, &target)
}

func CheckEmailReachable(email string) error {
	address, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return &EmailReachabilityError{Message: "邮箱格式不正确"}
	}
	parts := strings.Split(address.Address, "@")
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return &EmailReachabilityError{Message: "邮箱格式不正确"}
	}
	domain := strings.ToLower(strings.TrimSpace(parts[1]))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolver := net.DefaultResolver
	if records, err := resolver.LookupMX(ctx, domain); err == nil && len(records) > 0 {
		return nil
	}
	if hosts, err := resolver.LookupHost(ctx, domain); err == nil && len(hosts) > 0 {
		return nil
	}
	return &EmailReachabilityError{Message: fmt.Sprintf("邮箱域名 %s 不可达，请检查邮箱地址", domain)}
}
