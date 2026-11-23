package source

import (
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/godbus/dbus/v5"
)

func newTestSource() *linuxSource {
	return New(Options{Log: log.New(io.Discard, "", 0)}).(*linuxSource)
}

func notify(serial uint32, app string, replaces uint32) event {
	return event{typ: dbus.TypeMethodCall, member: "Notify", sender: ":1.50", serial: serial,
		body: []any{app, replaces, "", "summary", "body", []string{}, map[string]dbus.Variant{}, int32(-1)}}
}

func reply(toSerial, id uint32) event {
	return event{typ: dbus.TypeMethodReply, destination: ":1.50", replySerial: fmt.Sprint(toSerial), body: []any{id}}
}

func closed(id, reason uint32) event {
	return event{typ: dbus.TypeSignal, member: "NotificationClosed", body: []any{id, reason}}
}

func update(uri string, props map[string]dbus.Variant) event {
	return event{typ: dbus.TypeSignal, member: "Update", body: []any{uri, props}}
}

func TestLinuxNotificationTally(t *testing.T) {
	s := newTestSource()
	for _, e := range []event{
		notify(10, "Slack", 0), reply(10, 7),
		notify(11, "Firefox", 0), reply(11, 8),
		notify(12, "slack", 0), reply(12, 9),
	} {
		s.apply(e)
	}
	if c := s.count(); c != 2 {
		t.Fatalf("count = %d, want 2", c)
	}
	s.apply(notify(13, "Slack", 9)) // replaces 9: still 2
	s.apply(reply(13, 9))
	s.apply(closed(7, 1)) // expired into the tray: still unread
	if c := s.count(); c != 2 {
		t.Fatalf("after replace/expire count = %d, want 2", c)
	}
	s.apply(closed(7, 2)) // dismissed
	s.apply(closed(9, 3)) // closed by Slack
	if c := s.count(); c != 0 {
		t.Fatalf("after close count = %d, want 0", c)
	}
}

func TestLinuxLauncherBadgeWins(t *testing.T) {
	s := newTestSource()
	s.apply(notify(10, "Slack", 0))
	s.apply(reply(10, 7))
	s.apply(update("application://firefox.desktop", map[string]dbus.Variant{"count": dbus.MakeVariant(int64(5))}))
	if c := s.count(); c != 1 {
		t.Fatalf("other app's badge changed count to %d", c)
	}
	s.apply(update("application://slack.desktop", map[string]dbus.Variant{"count": dbus.MakeVariant(int64(4)), "count-visible": dbus.MakeVariant(true)}))
	if c := s.count(); c != 4 {
		t.Fatalf("badge count = %d, want 4", c)
	}
	s.apply(update("application://slack.desktop", map[string]dbus.Variant{"count-visible": dbus.MakeVariant(false)}))
	if c := s.count(); c != 0 {
		t.Fatalf("hidden badge count = %d, want 0", c)
	}
}
