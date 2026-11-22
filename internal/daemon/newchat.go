package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"

	"github.com/MarcFord/gmessages-omarchy-plugin/internal/wire"
)

// Contacts returns the phone's contacts that have a number, for the
// new-message picker. The plugin filters them as you type.
func (d *Daemon) Contacts(ctx context.Context) ([]wire.Contact, error) {
	c, err := d.requireClient()
	if err != nil {
		return nil, err
	}
	resp, err := withAuthRetry(d, func() (*gmproto.ListContactsResponse, error) {
		return c.ListContacts(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	out := make([]wire.Contact, 0, len(resp.GetContacts()))
	for _, ct := range resp.GetContacts() {
		num := ct.GetNumber().GetNumber()
		if num == "" {
			continue
		}
		out = append(out, wire.Contact{Name: ct.GetName(), Number: num})
	}
	return out, nil
}

// NewConversation finds or creates the one-to-one thread with a number and
// adds it to the conversation list, so the plugin can select it right away.
func (d *Daemon) NewConversation(ctx context.Context, p wire.NewConversationParams) (wire.Conversation, error) {
	number := strings.TrimSpace(p.Number)
	if number == "" {
		return wire.Conversation{}, errors.New("no number given")
	}
	c, err := d.requireClient()
	if err != nil {
		return wire.Conversation{}, err
	}
	resp, err := withAuthRetry(d, func() (*gmproto.GetOrCreateConversationResponse, error) {
		return c.GetOrCreateConversation(ctx, &gmproto.GetOrCreateConversationRequest{
			Numbers: []*gmproto.ContactNumber{{
				// Same values the mautrix bridge sends for a one-to-one chat.
				MysteriousInt: 2,
				Number:        number,
				Number2:       number,
			}},
		})
	})
	if err != nil {
		return wire.Conversation{}, fmt.Errorf("start conversation: %w", err)
	}
	if resp.GetConversation() == nil {
		return wire.Conversation{}, fmt.Errorf("start conversation: no conversation in response (status %s)", resp.GetStatus())
	}
	d.upsertConversation(resp.GetConversation())
	return convertConversation(resp.GetConversation()), nil
}
