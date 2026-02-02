package srv

import (
	"errors"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestDealWithWildcards(t *testing.T) {
	// Mock function to simulate MariaDB MAX(uid) or COUNT(*)
	mockGetNext := func() (uint32, error) {
		return 100, nil
	}

	t.Run("Resolve SeqSet Wildcard", func(t *testing.T) {
		// 1:* represents Start: 1, Stop: 0
		set := imap.SeqSet{{Start: 1, Stop: 0}}

		err := dealWithWildcards(set, mockGetNext)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if set[0].Stop != 100 {
			t.Errorf("Expected Stop to be 100, got %d", set[0].Stop)
		}
	})

	t.Run("Resolve UIDSet Wildcard", func(t *testing.T) {
		// UID 50:*
		set := imap.UIDSet{{Start: 50, Stop: 0}}

		err := dealWithWildcards(set, mockGetNext)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if set[0].Stop != 100 {
			t.Errorf("Expected Stop to be 100, got %d", set[0].Stop)
		}
	})

	t.Run("Error from getNextItem", func(t *testing.T) {
		set := imap.SeqSet{{Start: 1, Stop: 0}}
		failNext := func() (uint32, error) {
			return 0, errors.New("db error")
		}

		err := dealWithWildcards(set, failNext)
		if err == nil || err.Error() != "db error" {
			t.Errorf("Expected 'db error', got %v", err)
		}
	})

	t.Run("Static Set Returns Nil", func(t *testing.T) {
		// 1:10 (no zeros)
		set := imap.SeqSet{{Start: 1, Stop: 10}}
		err := dealWithWildcards(set, mockGetNext)
		if err != nil {
			t.Errorf("Expected nil for static set, got %v", err)
		}
	})
}
