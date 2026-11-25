package daemon

import (
	"github.com/rs/zerolog"

	"github.com/MarcFord/gmessages-omarchy-plugin/internal/wire"
)

// The phone can drop this pairing's encryption keys on its own (an app update,
// cleared data, a revoke) while Google's side stays perfectly happy. Requests
// still authenticate and the phone still sends updates, but every one of them
// fails to decrypt and the phone never answers a request again. libgm only
// logs that failure; it raises no event and never recovers. Left alone the
// daemon sits "connected" with an empty panel indefinitely.
//
// So the daemon watches libgm's log for the failure. A healthy session
// decrypts something between occasional bad messages; a run of failures with
// nothing decrypted in between means the keys are dead, and only a new pairing
// replaces them.

// decryptFailMsg is the exact message libgm logs when an incoming RPC message
// cannot be decrypted or decoded (pkg/libgm/event_handler.go, HandleRPCMsg).
const decryptFailMsg = "Failed to decode incoming RPC message"

// decryptFailThreshold counts failures with no successful decrypt between
// them. A reconnect against dead keys produces a burst of about four, so this
// is two bursts: one stray bad message never trips it.
const decryptFailThreshold = 8

// decryptWatch sees every libgm log line and reports decrypt failures.
type decryptWatch struct{ d *Daemon }

func (w decryptWatch) Run(_ *zerolog.Event, level zerolog.Level, msg string) {
	if level == zerolog.ErrorLevel && msg == decryptFailMsg {
		w.d.noteDecryptFailure()
	}
}

// noteDecryptOK records that the phone's data decrypted, so the keys work.
func (d *Daemon) noteDecryptOK() {
	d.mu.Lock()
	d.decryptFails = 0
	d.mu.Unlock()
}

func (d *Daemon) noteDecryptFailure() {
	d.mu.Lock()
	d.decryptFails++
	trip := d.decryptFails >= decryptFailThreshold && d.paired && !d.keyRepairing
	if trip {
		d.decryptFails = 0
		d.keyRepairing = true
	}
	d.mu.Unlock()
	if trip {
		go d.recoverFromKeyMismatch()
	}
}

// recoverFromKeyMismatch re-pairs, which negotiates fresh keys. When that
// cannot finish, the stored session is worthless (its keys will never decrypt
// again), so it is dropped and the panel returns to the pairing screen instead
// of pretending to be connected.
func (d *Daemon) recoverFromKeyMismatch() {
	defer func() {
		d.mu.Lock()
		d.keyRepairing = false
		d.mu.Unlock()
	}()

	d.log.Warn().Msg("Phone's messages no longer decrypt; the pairing is dead, re-pairing")
	if d.repairSession() {
		return
	}

	d.mu.Lock()
	d.paired = false
	state := d.status.State
	d.mu.Unlock()
	_ = d.paths.ClearSession()

	// PairFromBrowser leaves a specific hint (which profile, which cookie) in
	// the error state; keep it rather than replacing it with a generic one.
	if state == wire.StateError {
		return
	}
	d.setState(wire.StateUnpaired, "your phone no longer recognises this pairing — pair again")
}
