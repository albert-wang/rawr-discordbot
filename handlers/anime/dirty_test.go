package anime

import "testing"

func newTestDB(names ...string) *Database {
	db := &Database{animes: map[string]Status{}, dirty: map[string]struct{}{}}
	for _, n := range names {
		db.animes[n] = Status{Name: n, CurrentEpisode: 9}
	}
	return db
}

func assertDirty(t *testing.T, db *Database, want ...string) {
	t.Helper()
	if len(db.dirty) != len(want) {
		t.Fatalf("dirty = %v, want exactly %v", db.dirty, want)
	}
	for _, w := range want {
		if _, ok := db.dirty[w]; !ok {
			t.Fatalf("dirty = %v, missing %q", db.dirty, w)
		}
	}
}

func TestSetMarksOnlyTouchedRowDirty(t *testing.T) {
	db := newTestDB("foo", "bar")

	if _, err := db.Set(&SetArguments{Anime: "foo", Episode: 10}); err != nil {
		t.Fatal(err)
	}

	assertDirty(t, db, "foo")
	if got := db.animes["foo"].CurrentEpisode; got != 10 {
		t.Fatalf("foo episode = %d, want 10", got)
	}
	if got := db.animes["bar"].CurrentEpisode; got != 9 {
		t.Fatalf("bar episode = %d, want untouched 9", got)
	}
}

func TestSetNewAnimeMarksDirty(t *testing.T) {
	db := newTestDB("foo")

	if _, err := db.Set(&SetArguments{Anime: "baz", Episode: 1}); err != nil {
		t.Fatal(err)
	}

	assertDirty(t, db, "baz")
}

func TestSubSrcDeleteMarkDirty(t *testing.T) {
	db := newTestDB("foo", "bar", "qux")

	if _, err := db.Sub(&SubArguments{Anime: "foo", Subber: "SubsPlease"}); err != nil {
		t.Fatal(err)
	}
	src := "cr://example"
	if _, err := db.Src(&SrcArguments{Anime: "bar", Source: &src}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Delete(&DeleteArguments{Name: "qux"}); err != nil {
		t.Fatal(err)
	}

	assertDirty(t, db, "foo", "bar", "qux")
}

func TestReadOnlyCommandsLeaveNothingDirty(t *testing.T) {
	db := newTestDB("foo", "bar")

	if _, err := db.List(&ListArguments{}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Inspect(&InspectArguments{Anime: "foo"}); err != nil {
		t.Fatal(err)
	}
	// Src with no new source and no --remove is a lookup, not a write.
	if _, err := db.Src(&SrcArguments{Anime: "foo"}); err != nil {
		t.Fatal(err)
	}
	// Unknown anime must not create or dirty anything.
	if _, err := db.Sub(&SubArguments{Anime: "nope", Subber: "x"}); err != nil {
		t.Fatal(err)
	}

	assertDirty(t, db)
}
