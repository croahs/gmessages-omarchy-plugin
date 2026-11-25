package daemon

import (
	"testing"

	"github.com/rs/zerolog"
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
