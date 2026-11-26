package daemon

import (
	"testing"

	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
)

func TestDecryptWatchCountsOnlyDecodeFailures(t *testing.T) {
	d := &Daemon{}
	// zerolog.Nop() skips hooks entirely; a nil writer discards but runs them.
	log := zerolog.New(nil).Hook(decryptWatch{d})

	log.Error().Msg(decryptFailMsg)
	log.Warn().Msg(decryptFailMsg)
	log.Error().Msg("something else")
	if d.decryptFails != 1 {
		t.Fatalf("decryptFails = %d, want 1", d.decryptFails)
	}

	d.noteDecryptOK()
	if d.decryptFails != 0 {
		t.Fatalf("decryptFails after OK = %d, want 0", d.decryptFails)
	}
}

func TestDecryptFailuresTripOnceWhenPaired(t *testing.T) {
	// keyRepairing pre-set stands in for a recovery already running, so the
	// test never starts one against the real browser.
	d := &Daemon{paired: true, keyRepairing: true}
	for i := 0; i < decryptFailThreshold*2; i++ {
		d.noteDecryptFailure()
	}
	if d.decryptFails != decryptFailThreshold*2 {
		t.Fatalf("counter reset while a recovery was running: %d", d.decryptFails)
	}
}

func TestDeadKeysAreNotResumed(t *testing.T) {
	// A key-mismatch re-pair that fails must not fall back to the stored
	// session: its keys are dead, so resuming would trip the watch again.
	d := &Daemon{client: &libgm.Client{}, paired: true, keyRepairing: true}
	if d.resumeSession() {
		t.Fatal("resumed a session whose keys no longer decrypt")
	}
}
