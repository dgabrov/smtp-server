package queue

import (
	"cmp"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

func ProcItem(from string, to string, message *string, tlsConfig *tls.Config, localDomain string, port int) error {

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
		slog.Info(fmt.Sprintf("did not find mx entry for the domain %s deliver to default", domain))
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
		slog.Info(fmt.Sprintf("id: %s delivering from %s to %s, with mx: %s", id, from, to, deliveryAddress))
		err = deliverMail(from, to, message, deliveryAddress, tlsConfig, localDomain)

		if err == nil {
			slog.Info("processed ok the item with the id: ", id)
			// worked, no need to send with the next mx address
			return nil
		} else {
			slog.Error(fmt.Sprintf("error for id: %s, error: %v", id, err))
		}
	}

	return err // return the latest error that happens
}

func deliverMail(from string, to string, message *string, mxAddress string, tlsConfig *tls.Config, localDomain string) error {
	client, err := smtp.DialStartTLS(mxAddress, tlsConfig)
	if err != nil {
		return err
	}
	defer client.Close()

	err = client.Hello(localDomain)
	if err != nil {
		return err
	}

	return client.SendMail(from, []string{to}, strings.NewReader(*message))
}

func getDomain(addr string) (string, error) {
	items := strings.Split(addr, "@")
	if len(items) != 2 {
		return "", fmt.Errorf("invalid address: %s", addr)
	}

	return items[1], nil
}
