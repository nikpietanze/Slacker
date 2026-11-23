package source

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// On Linux we passively monitor the user's session D-Bus:
//
//   - org.freedesktop.Notifications.Notify calls whose app name contains
//     "slack" add a notification; NotificationClosed with reason "dismissed"
//     or "closed by the app" (Slack closes them as you read) removes it.
//     Popups that merely time out stay counted, as they do in the tray.
//   - If Slack publishes a launcher badge (com.canonical.Unity.LauncherEntry,
//     used by Electron apps under KDE, Ubuntu Dock, Dash to Dock...), that
//     count is Slack's own and wins over the notification tally.
//
// Monitoring the session bus is allowed for the bus owner (you) without root.

type linuxSource struct {
	opts Options

	mu        sync.Mutex
	pending   map[string]bool // "sender/serial" of Slack Notify calls awaiting their reply
	active    map[uint32]bool // ids of Slack notifications still open
	badge     int
	haveBadge bool
}

// New returns the platform source.
func New(opts Options) Source {
	return &linuxSource{opts: opts, pending: map[string]bool{}, active: map[uint32]bool{}}
}

func (s *linuxSource) Name() string { return "Linux D-Bus notifications" }

func (s *linuxSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.haveBadge {
		return s.badge
	}
	return len(s.active)
}

func (s *linuxSource) monitor() (*dbus.Conn, chan *dbus.Message, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, nil, fmt.Errorf("connect session bus: %w", err)
	}
	returnRule := "type='method_return'"
	var owner string
	if err := conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, "org.freedesktop.Notifications").Store(&owner); err == nil && owner != "" {
		returnRule += ",sender='" + owner + "'"
	}
	rules := []string{
		"type='method_call',interface='org.freedesktop.Notifications',member='Notify'",
		"type='signal',interface='org.freedesktop.Notifications',member='NotificationClosed'",
		"type='signal',interface='com.canonical.Unity.LauncherEntry',member='Update'",
		returnRule,
	}
	call := conn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, uint32(0))
	if call.Err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("become monitor: %w", call.Err)
	}
	ch := make(chan *dbus.Message, 64)
	conn.Eavesdrop(ch)
	return conn, ch, nil
}

// closeMonitor detaches the eavesdrop channel before closing; closing with it
// attached can race godbus's reader into a send on a closed channel.
func closeMonitor(conn *dbus.Conn) {
	conn.Eavesdrop(nil)
	conn.Close()
}

func (s *linuxSource) Run(ctx context.Context, report func(int)) error {
	for {
		conn, ch, err := s.monitor()
		if err != nil {
			return err
		}
		report(s.count())
		alive := time.NewTicker(30 * time.Second)
		done := false
		for !done {
			select {
			case <-ctx.Done():
				alive.Stop()
				closeMonitor(conn)
				return nil
			case <-alive.C:
				done = !conn.Connected()
			case msg, ok := <-ch:
				if !ok {
					done = true // bus connection dropped; reconnect
					break
				}
				if s.handle(msg) {
					report(s.count())
				}
			}
		}
		alive.Stop()
		closeMonitor(conn)
		s.opts.Log.Printf("session bus connection lost; reconnecting")
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(3 * time.Second):
		}
	}
}

// event is the part of a D-Bus message we care about.
type event struct {
	typ         dbus.Type
	member      string
	sender      string // unique name of the sender
	serial      uint32
	destination string
	replySerial string
	body        []any
}

func eventOf(msg *dbus.Message) event {
	header := func(f dbus.HeaderField) string {
		if v, ok := msg.Headers[f]; ok {
			return fmt.Sprint(v.Value())
		}
		return ""
	}
	return event{
		typ:         msg.Type,
		member:      header(dbus.FieldMember),
		sender:      header(dbus.FieldSender),
		serial:      msg.Serial(),
		destination: header(dbus.FieldDestination),
		replySerial: header(dbus.FieldReplySerial),
		body:        msg.Body,
	}
}

func (s *linuxSource) handle(msg *dbus.Message) bool { return s.apply(eventOf(msg)) }

// apply updates state from one message and reports whether it changed.
func (s *linuxSource) apply(msg event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch msg.typ {
	case dbus.TypeMethodCall:
		if msg.member != "Notify" || len(msg.body) < 2 {
			return false
		}
		app, _ := msg.body[0].(string)
		if !strings.Contains(strings.ToLower(app), "slack") {
			return false
		}
		if replaces, _ := msg.body[1].(uint32); replaces != 0 && s.active[replaces] {
			return false // updating a notification we already count
		}
		s.pending[msg.sender+"/"+fmt.Sprint(msg.serial)] = true
		if len(s.pending) > 256 { // replies we never saw; don't grow forever
			s.pending = map[string]bool{}
		}
		s.opts.debugf("slack notification requested")
	case dbus.TypeMethodReply:
		key := msg.destination + "/" + msg.replySerial
		if !s.pending[key] {
			return false
		}
		delete(s.pending, key)
		if len(msg.body) > 0 {
			if id, ok := msg.body[0].(uint32); ok {
				s.active[id] = true
				return true
			}
		}
	case dbus.TypeSignal:
		switch msg.member {
		case "NotificationClosed":
			if len(msg.body) < 2 {
				return false
			}
			id, _ := msg.body[0].(uint32)
			reason, _ := msg.body[1].(uint32)
			// 1 = expired (moves to the tray, still unread), 2 = dismissed by
			// the user, 3 = closed by the app, 4 = undefined.
			if s.active[id] && reason != 1 {
				delete(s.active, id)
				return true
			}
		case "Update":
			if len(msg.body) < 2 {
				return false
			}
			uri, _ := msg.body[0].(string)
			props, _ := msg.body[1].(map[string]dbus.Variant)
			if !strings.Contains(strings.ToLower(uri), "slack") || props == nil {
				return false
			}
			n, visible := s.badge, true
			if v, ok := props["count"]; ok {
				switch c := v.Value().(type) {
				case int64:
					n = int(c)
				case int32:
					n = int(c)
				case uint32:
					n = int(c)
				}
			}
			if v, ok := props["count-visible"]; ok {
				visible, _ = v.Value().(bool)
			}
			if !visible {
				n = 0
			}
			s.badge, s.haveBadge = n, true
			s.opts.debugf("slack launcher badge: %d", n)
			return true
		}
	}
	return false
}

func (s *linuxSource) Diagnose(ctx context.Context) string {
	var b strings.Builder
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintf(&b, "session bus: %v\n", err)
		return b.String()
	}
	defer conn.Close()
	var owner string
	if err := conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, "org.freedesktop.Notifications").Store(&owner); err != nil {
		fmt.Fprintf(&b, "notification server: not found (%v)\n", err)
	} else {
		fmt.Fprintf(&b, "notification server: %s\n", owner)
	}
	mon, _, err := s.monitor()
	if err != nil {
		fmt.Fprintf(&b, "bus monitoring: FAILED (%v)\n", err)
	} else {
		closeMonitor(mon)
		b.WriteString("bus monitoring: ok\n")
	}
	return b.String()
}
