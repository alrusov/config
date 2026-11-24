package config

import (
	"testing"
)

//----------------------------------------------------------------------------------------------------------------------------//

func TestAuthSlice2MapEmptyElements(t *testing.T) {
	src := map[string][]string{
		"/a": {"user1", "", "  ", "!user2", "!", "@g1"},
		"/b": {""},
	}

	dst := authSlice2Map(src)

	a := dst["/a"]
	if len(a) != 3 {
		t.Fatalf(`/a: expected 3 elements, got %d: %v`, len(a), a)
	}
	if v, ok := a["user1"]; !ok || !v {
		t.Errorf(`/a: user1 must be allowed, got %v %v`, v, ok)
	}
	if v, ok := a["user2"]; !ok || v {
		t.Errorf(`/a: user2 must be denied, got %v %v`, v, ok)
	}
	if v, ok := a["@g1"]; !ok || !v {
		t.Errorf(`/a: @g1 must be allowed, got %v %v`, v, ok)
	}

	if len(dst["/b"]) != 0 {
		t.Errorf(`/b: expected empty list, got %v`, dst["/b"])
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
