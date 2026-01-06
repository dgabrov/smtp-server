package queue

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/dgb9/smtp-server/internal/srv"
)

func StartQueue(wg *sync.WaitGroup, config data.QueueConfig, server srv.Servr) {
	loaderChannel := make(chan *data.DmQueueRecipient, 1)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	go startLoader(ctx, config, server, loaderChannel)

	for i := 0; i < config.SimultaneousProcessing; i++ {
		go processQueue(ctx, server, loaderChannel, config.TimeBetweenAttempts)
	}

	wg.Done()
}

func startLoader(ctx context.Context, config data.QueueConfig, server srv.Servr, channel chan *data.DmQueueRecipient) {
	for {
		queue, err := server.LoadQueueRecipients(ctx, config)

		if err != nil {
			slog.Error("there was error loading the slice in the queue: ", err.Error())
		} else {
			ln := len(queue)
			slog.Info(fmt.Sprintf("loaded %d items from the queue", ln))
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
			slog.Info("processing terminated, stopping loader")
			break
		case <-time.After(time.Duration(config.TimeBetweenLoads) * time.Second):
			slog.Info("load next one")
		}
	}
}

func processQueue(ctx context.Context, server srv.Servr, channel chan *data.DmQueueRecipient, errorDelay int) {
	for {
		var item *data.DmQueueRecipient
		select {
		case <-ctx.Done():
			slog.Info("processing terminated")
			break
		case item = <-channel:
			slog.Info(fmt.Sprintf("processing item: ", item.QueueRecipientID))
		}

		// processing an item
		queue, err := server.LoadQueueItemByID(ctx, item.QueueID)
		if err != nil {
			slog.Error("there was error loading the queue item: ", err.Error())
			_ = server.AddQueueItemError(ctx, item.QueueRecipientID)
		}

		// let's process them
		err = processMailQueueItem(queue, item) // this does mx all the stuff there
		if err != nil {
			_ = server.AddQueueItemError(ctx, item.QueueRecipientID)
		} else {
			_ = server.MarkQueueItemSuccess(ctx, item.QueueRecipientID)
		}
	}
}

func processMailQueueItem(queue *data.DmQueue, item *data.DmQueueRecipient) error {
	// this is the big stuff, will be in separate file
	slog.Info(fmt.Sprintf("processing ok the item with the id: %s for the queue id: %s", item.QueueRecipientID, item.QueueID))

	return nil
}
