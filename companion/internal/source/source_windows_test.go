package source

import "testing"

func TestWindowsAckSurvivesIDReuse(t *testing.T) {
	s := &windowsSource{acked: map[int64]bool{}}
	ids := []int64{10, 11}
	s.pruneAcked(ids)
	if n := s.unacked(ids); n != 2 {
		t.Fatalf("unacked = %d, want 2", n)
	}
	for _, id := range ids { // Slack focused
		s.acked[id] = true
	}
	if n := s.unacked(ids); n != 0 {
		t.Fatalf("after focus unacked = %d, want 0", n)
	}
	ids = []int64{10} // toast 11 removed from the notification center
	s.pruneAcked(ids)
	ids = []int64{10, 11} // a new toast reuses id 11
	s.pruneAcked(ids)
	if n := s.unacked(ids); n != 1 {
		t.Fatalf("reused id: unacked = %d, want 1", n)
	}
}
