package queue

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/google/uuid"
)

func StartQueue(config data.QueueConfig, server srv.Servr, tlsConfig *tls.Config, localDomain string, dkimConfig data.DkimConfig, dkimKey *rsa.PrivateKey) {
	loaderChannel := make(chan *data.DmQueueRecipient, 1)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	go startLoader(ctx, config, server, loaderChannel)

	for i := 0; i < config.SimultaneousProcessing; i++ {
		go processQueue(ctx, server, loaderChannel, tlsConfig, localDomain, dkimConfig, dkimKey)
	}
}

func startLoader(ctx context.Context, config data.QueueConfig, server srv.Servr, channel chan *data.DmQueueRecipient) {
	for {
		queue, err := server.LoadQueueRecipients(ctx, config)

		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("there was error loading the slice in the queue: %s", err.Error()))
		} else {
			ln := len(queue)
			slog.InfoContext(ctx, fmt.Sprintf("loaded %d items from the queue", ln))
			if ln > 0 {
				// data existent, push it into channels
				for _, item := range queue {
					channel <- item
				}
			}
		}

		// if not, wait the timeout
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "processing terminated, stopping loader")
			return
		case <-time.After(time.Duration(config.TimeBetweenLoads) * time.Second):
			slog.InfoContext(ctx, "load next one")
		}
	}
}

func processQueue(ctx context.Context, server srv.Servr, channel chan *data.DmQueueRecipient, tlsConfig *tls.Config, localDomain string, dkimConfig data.DkimConfig, dkimKey *rsa.PrivateKey) {
	for {
		var item *data.DmQueueRecipient
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "processing terminated")
			break
		case item = <-channel:
			slog.InfoContext(ctx, fmt.Sprintf("processing item: %s", item.QueueRecipientID))
		}

		// if item pointer is null, exit, that means channel is closed
		if item == nil {
			slog.InfoContext(ctx, "exit queue processing after channel closed")
			break
		}

		// processing an item
		queue, err := server.LoadQueueItemByID(ctx, item.QueueID)
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("there was error loading the queue item: %s", err.Error()))
			_ = server.AddQueueItemError(ctx, item.QueueRecipientID)
		}

		// let's process them
		err = processMailQueueItem(queue, item, tlsConfig, localDomain, dkimConfig, dkimKey) // this does mx all the stuff there

		if err != nil {
			_ = server.AddQueueItemError(ctx, item.QueueRecipientID)
		} else {
			_ = server.MarkQueueItemSuccess(ctx, item.QueueRecipientID)
		}
	}
}

func processMailQueueItem(queue *data.DmQueue, item *data.DmQueueRecipient, tlsConfig *tls.Config, localDomain string, config data.DkimConfig, dkimKey *rsa.PrivateKey) error {
	ctx := context.WithValue(context.Background(), "transaction-id", uuid.NewString())

	return ProcItem(ctx, queue.From, item.ToAddr, &queue.Body, tlsConfig, localDomain, 25, config.Enabled, config.Selector, dkimKey)
}
