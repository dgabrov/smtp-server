package srv

import (
	"fmt"
	"strings"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-imap/v2"
)

func filterPositionalMessages(res []*data.DmPositionalMessage, set imap.NumSet) []*data.DmPositionalMessage {
	var filtered []*data.DmPositionalMessage

	seqSet, ok := set.(imap.SeqSet)
	if ok {
		for _, item := range res {
			seq := item.SeqNum

			if containsSeq(&seqSet, seq) {
				filtered = append(filtered, item)
			}
		}
	}

	uidSet, ok := set.(imap.UIDSet)
	if ok {
		for _, item := range res {
			uid := imap.UID(item.UID)

			if containsUid(&uidSet, uid) {
				filtered = append(filtered, item)
			}
		}
	}

	return filtered
}

func filterSeqHolders(items []*data.SeqHolder, uids *imap.UIDSet) []*data.SeqHolder {
	var result []*data.SeqHolder
	if uids == nil {
		result = append(result, items...)

		return result
	}

	// we filter
	for _, item := range items {
		uid := imap.UID(item.UID)

		if containsUid(uids, uid) {
			result = append(result, item)
		}
	}
	return result
}

func containsSeq(set *imap.SeqSet, seq uint32) bool {
	for _, interval := range *set {
		start := interval.Start
		stop := interval.Stop

		if start == 0 && stop == 0 {
			// crazy, should not be both of them zero
			return true
		}

		if start == 0 || (start > stop && stop != 0) {
			man := start
			start = stop
			stop = man
		}

		// now start and stop are in order, we can compare
		if seq >= start && (stop == 0 || seq <= stop) {
			return true
		}
	}

	return false
}

func containsUid(uids *imap.UIDSet, uid imap.UID) bool {
	for _, interval := range *uids {
		start := interval.Start
		stop := interval.Stop

		if start == 0 && stop == 0 {
			// crazy, should not be both of them zero
			return true
		}

		if start == 0 || (start > stop && stop != 0) {
			man := start
			start = stop
			stop = man
		}

		// now start and stop are in order, we can compare
		if uid >= start && (stop == 0 || uid <= stop) {
			return true
		}
	}

	return false
}

func GetDomain(addr string) (string, error) {
	items := strings.Split(addr, "@")
	if len(items) != 2 {
		return "", fmt.Errorf("invalid address: %s", addr)
	}

	return items[1], nil
}
