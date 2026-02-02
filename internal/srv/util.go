package srv

import (
	"errors"

	"github.com/emersion/go-imap/v2"
)

func dealWithWildcards(numSet imap.NumSet, getNextItemNumSet func() (uint32, error), getNextItemUid func() (imap.UID, error)) error {
	if numSet == nil {
		return nil
	}

	var nextItem uint32 = 0
	var err error

	seqSet, ok := numSet.(imap.SeqSet)
	if ok {
		if seqSet.Dynamic() {
			ln := len(seqSet)
			for i := 0; i < ln; i++ {
				if seqSet[i].Stop == 0 {
					if nextItem == 0 {
						if getNextItemNumSet == nil {
							return errors.New("getNextItemNumSet is nil")
						}

						nextItem, err = getNextItemNumSet()

						if err != nil {
							return err
						}
					}

					seqSet[i].Stop = nextItem
				}
			}
		}

		return nil
	}

	uidSet, ok := numSet.(imap.UIDSet)
	var maxUid imap.UID = 0
	if ok {
		if uidSet.Dynamic() {
			ln := len(uidSet)
			for i := 0; i < ln; i++ {
				if uidSet[i].Stop == 0 {
					if maxUid == 0 {
						if getNextItemUid == nil {
							return errors.New("you did not pass the function for maximum uid")
						}
						maxUid, err = getNextItemUid()

						if err != nil {
							return err
						}
					}
				}

				uidSet[i].Stop = maxUid
			}
		}

		return nil
	}

	return errors.New("the passed construct is not either seqSet or uidSet")
}
