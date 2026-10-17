package session

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func testStore(t *testing.T) Store {
	t.Helper()
	dir := t.TempDir()
	return Store{Dir: filepath.Join(dir, "sessions"), Legacy: filepath.Join(dir, "session")}
}

// TestStoreRoundTripsNamedAccounts: each name is its own 0600 file, and the legacy path is the
// account called default.
func TestStoreRoundTripsNamedAccounts(t *testing.T) {
	s := testStore(t)
	if err := s.Save("work", Secret("a"), Secret("b")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Save(DefaultName, Secret("c"), Secret("d")); err != nil {
		t.Fatalf("Save default: %v", err)
	}
	work, err := s.Load("work")
	if err != nil || work.AuthToken != "a" {
		t.Fatalf("Load work = %+v, %v", work, err)
	}
	def, err := s.Load(DefaultName)
	if err != nil || def.AuthToken != "c" {
		t.Fatalf("Load default = %+v, %v", def, err)
	}
	if s.Path(DefaultName) != s.Legacy {
		t.Fatal("the default account is not the legacy file")
	}
	info, err := os.Stat(s.Path("work"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("named session mode %04o", perm)
	}
}

// TestStoreNames: default first when it exists, then the named files sorted; a directory that does
// not exist yet is an empty store, not an error.
func TestStoreNames(t *testing.T) {
	s := testStore(t)
	names, err := s.Names()
	if err != nil || len(names) != 0 {
		t.Fatalf("empty store = %v, %v", names, err)
	}
	if err := s.Save("beta", Secret("a"), Secret("b")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("alpha", Secret("a"), Secret("b")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(DefaultName, Secret("a"), Secret("b")); err != nil {
		t.Fatal(err)
	}
	names, err = s.Names()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{DefaultName, "alpha", "beta"}; !slices.Equal(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

// TestStoreRefusesNamesThatArePaths: a name is a file name, never a path.
func TestStoreRefusesNamesThatArePaths(t *testing.T) {
	s := testStore(t)
	for _, bad := range []string{"", "../escape", "a/b", ".hidden", "with space", "x:y"} {
		if ValidName(bad) {
			t.Fatalf("%q is accepted as a name", bad)
		}
		if s.Path(bad) != "" {
			t.Fatalf("%q resolved to a path", bad)
		}
		if err := s.Save(bad, "a", "b"); err == nil {
			t.Fatalf("%q was written", bad)
		}
	}
	if !ValidName("work-laptop_2") {
		t.Fatal("a reasonable name was refused")
	}
}

// TestStoreActivePrefersTheRequestedName: the config's choice wins, then default, then the first.
func TestStoreActivePrefersTheRequestedName(t *testing.T) {
	s := testStore(t)
	if _, ok, err := s.Active("work"); err != nil || ok {
		t.Fatalf("an empty store answered %v, %v", ok, err)
	}
	if err := s.Save("work", Secret("a"), Secret("b")); err != nil {
		t.Fatal(err)
	}
	if name, ok, _ := s.Active("work"); name != "work" || !ok {
		t.Fatalf("requested name = %q ok=%v", name, ok)
	}
	if name, ok, _ := s.Active("missing"); name != "work" || ok {
		t.Fatalf("fallback = %q ok=%v", name, ok)
	}
	if err := s.Save(DefaultName, Secret("a"), Secret("b")); err != nil {
		t.Fatal(err)
	}
	if name, _, _ := s.Active("missing"); name != DefaultName {
		t.Fatalf("fallback with a default = %q", name)
	}
}

// TestStoreDelete: removing a named account leaves the others.
func TestStoreDelete(t *testing.T) {
	s := testStore(t)
	if err := s.Save("work", Secret("a"), Secret("b")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("home", Secret("a"), Secret("b")); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("work"); err != nil {
		t.Fatal(err)
	}
	names, _ := s.Names()
	if len(names) != 1 || names[0] != "home" {
		t.Fatalf("names after delete = %v", names)
	}
	if err := s.Delete("work"); err != nil {
		t.Fatalf("deleting a missing account: %v", err)
	}
}
