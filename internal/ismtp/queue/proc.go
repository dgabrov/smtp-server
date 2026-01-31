package queue

import (
	"cmp"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

func ProcItem(ctx context.Context, from string, to string, message *string, tlsConfig *tls.Config, localDomain string, port int, dkimEnabled bool, dkimSelector string, dkimKey *rsa.PrivateKey) error {

	// get the to address and get the mx values
	domain, err := getDomain(to)
	if err != nil {
		return err
	}

	// get the mx entries
	mxAddresses, err := net.LookupMX(domain)
	if err != nil {
		return err
	}

	if len(mxAddresses) == 0 {
		slog.InfoContext(ctx, fmt.Sprintf("did not find mx entry for the domain %s deliver to default", domain))
		mxAddresses = append(mxAddresses, &net.MX{Host: domain, Pref: 0})
	}

	// sort the entries
	slices.SortFunc(mxAddresses, func(first, second *net.MX) int {
		return cmp.Compare(first.Pref, second.Pref)
	})

	for _, mxAddress := range mxAddresses {
		host := mxAddress.Host
		if strings.HasSuffix(host, ".") {
			host = host[:len(host)-1]
		}

		deliveryAddress := fmt.Sprintf("%s:%d", host, port)
		id := uuid.NewString()
		slog.InfoContext(ctx, fmt.Sprintf("id: %s delivering from %s to %s, with mx: %s", id, from, to, deliveryAddress))

		if dkimEnabled && message != nil {
			slog.InfoContext(ctx, fmt.Sprintf("processing enabled dkim for id: %s", id))
			signed, err := signMessage([]byte(*message), dkimKey, localDomain, dkimSelector)
			if err != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("error dkim processing for: %s, error: %v", id, err))
			} else {
				slog.InfoContext(ctx, fmt.Sprintf("dkim processing for id: %s successful", id))
				msg := string(signed)
				message = &msg
			}
		}

		err = deliverMail(ctx, from, to, message, deliveryAddress, tlsConfig, localDomain)

		if err == nil {
			slog.InfoContext(ctx, fmt.Sprintf("processed ok the item with the id: %s", id))
			// worked, no need to send with the next mx address
			return nil
		} else {
			slog.ErrorContext(ctx, fmt.Sprintf("error for id: %s, error: %v", id, err))
		}
	}

	return err // return the latest error that happens
}

func deliverMail(ctx context.Context, from string, to string, message *string, mxAddress string, tlsConfig *tls.Config, localDomain string) error {
	slog.InfoContext(ctx, "start client")
	client, err := smtp.DialStartTLS(mxAddress, tlsConfig)
	if err != nil {
		return err
	}
	defer client.Close()

	slog.InfoContext(ctx, "client started successfully")

	err = client.Hello(localDomain)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "after hello")

	err = client.SendMail(from, []string{to}, strings.NewReader(*message))

	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("error sending, %v", err))
	} else {
		slog.InfoContext(ctx, "send mail successfully")
	}

	return err
}

func getDomain(addr string) (string, error) {
	items := strings.Split(addr, "@")
	if len(items) != 2 {
		return "", fmt.Errorf("invalid address: %s", addr)
	}

	return items[1], nil
}
