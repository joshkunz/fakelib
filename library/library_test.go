package library

import (
	"bytes"
	"log"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/google/go-cmp/cmp"
)

type trackInfo struct {
	Artist, Album, Title, Track string
}

func songInfo(song Song) (trackInfo, error) {
	t, err := id3v2.ParseReader(bytes.NewReader(song.tag), id3v2.Options{Parse: true})
	if err != nil {
		return trackInfo{}, err
	}
	return trackInfo{
		Artist: t.Artist(),
		Album:  t.Album(),
		Title:  t.Title(),
		Track:  t.GetTextFrame(t.CommonID("Track number/Position in set")).Text,
	}, nil
}

var testLibrary *Library

func init() {
	var err error
	testLibrary, err = New(bytes.NewReader(nil))
	if err != nil {
		log.Fatalf("failed to load library: %v", err)
	}
}

var libraryTests = []struct {
	idx          int
	wantLocation string
	wantInfo     trackInfo
}{
	{
		idx:          0,
		wantLocation: "A/A/A.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "A",
			Title:  "A",
			Track:  "1",
		},
	},
	{
		idx:          1,
		wantLocation: "A/A/B.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "A",
			Title:  "B",
			Track:  "2",
		},
	},
	{
		idx:          2,
		wantLocation: "A/A/C.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "A",
			Title:  "C",
			Track:  "3",
		},
	},
	{
		idx:          10,
		wantLocation: "A/B/A.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "B",
			Title:  "A",
			Track:  "1",
		},
	},
	{
		idx:          11,
		wantLocation: "A/B/B.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "B",
			Title:  "B",
			Track:  "2",
		},
	},
	{
		idx:          30,
		wantLocation: "B/A/A.mp3",
		wantInfo: trackInfo{
			Artist: "B",
			Album:  "A",
			Title:  "A",
			Track:  "1",
		},
	},
	{
		idx:          31,
		wantLocation: "B/A/B.mp3",
		wantInfo: trackInfo{
			Artist: "B",
			Album:  "A",
			Title:  "B",
			Track:  "2",
		},
	},
	{
		idx:          40,
		wantLocation: "B/B/A.mp3",
		wantInfo: trackInfo{
			Artist: "B",
			Album:  "B",
			Title:  "A",
			Track:  "1",
		},
	},
	{
		// 26 possible artists, 3 albums per artist, 10 tracks/album
		idx:          26 * 3 * 10,
		wantLocation: "AA/A/A.mp3",
		wantInfo: trackInfo{
			Artist: "AA",
			Album:  "A",
			Title:  "A",
			Track:  "1",
		},
	},
	{
		idx:          (26 * 3 * 10) + 1,
		wantLocation: "AA/A/B.mp3",
		wantInfo: trackInfo{
			Artist: "AA",
			Album:  "A",
			Title:  "B",
			Track:  "2",
		},
	},
}

func TestPathAt(t *testing.T) {
	for _, test := range libraryTests {
		loc, err := testLibrary.PathAt(test.idx)
		if err != nil {
			t.Errorf("testLibrary.PathAt(%d) = _, %v; want _, nil", test.idx, err)
			continue
		}
		if loc != test.wantLocation {
			t.Errorf("testLibrary.PathAt(%d) = %q, _; want %q, _", test.idx, loc, test.wantLocation)
		}
	}
}

func TestSongAt(t *testing.T) {
	for _, test := range libraryTests {
		song, err := testLibrary.SongAt(test.idx)
		if err != nil {
			t.Errorf("testLibrary.EntryAt(%d) = _, %v; want _, nil", test.idx, err)
			continue
		}

		info, err := songInfo(song)
		if err != nil {
			t.Errorf("failed to parse song at idx %d: %v", test.idx, err)
			continue
		}

		if diff := cmp.Diff(test.wantInfo, info); diff != "" {
			t.Errorf("testLibrary.SongAt(%d) diff in parsed song tag (want -> got):\n%s", test.idx, diff)
		}
	}
}

func TestLetterName(t *testing.T) {
	tests := []struct {
		idx  int
		want string
	}{
		{idx: 0, want: "A"},
		{idx: 25, want: "Z"},
		{idx: 26, want: "AA"},
		{idx: 27, want: "AB"},
		{idx: 26 + 25, want: "AZ"},
		{idx: 26 + (26 * 25), want: "ZA"},
		{idx: 26 + (26 * 25) + 25, want: "ZZ"},
		{idx: 26 + (26 * 26), want: "AAA"},
		{idx: 26 + (26 * 26) + 1, want: "AAB"},
	}

	for _, test := range tests {
		got := letterName(test.idx)
		if got != test.want {
			t.Errorf("letterName(%d) = %q, want %q", test.idx, got, test.want)
		}
	}
}

func TestCustomTagger(t *testing.T) {
	want := trackInfo{
		Artist: "Custom Artist",
		Album:  "Custom Album",
		Title:  "Custom Title",
	}

	tagF := func(idx int) *id3v2.Tag {
		t := id3v2.NewEmptyTag()
		t.SetArtist(want.Artist)
		t.SetAlbum(want.Album)
		t.SetTitle(want.Title)
		return t
	}

	lib, err := New(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Failed to create new library: %v", err)
	}
	lib.Tagger = tagF

	gotSong, err := lib.SongAt(0)
	if err != nil {
		t.Fatalf("lib.SongAt(0) = %v", err)
	}
	got, err := songInfo(gotSong)
	if err != nil {
		t.Fatalf("songInfo(...) = _, %v, want _, nil", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("lib.SongAt(0) had unexpected diff (want -> got):\n%s", diff)
	}
}

func TestCustomPather(t *testing.T) {
	wantTag := id3v2.NewEmptyTag()
	const want = "abc.mp3"

	lib, err := New(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Failed to create new library: %v", err)
	}
	lib.Tagger = func(int) *id3v2.Tag {
		return wantTag
	}
	lib.Pather = func(_ int, gotTag *id3v2.Tag) string {
		// Need to make sure that the pather is passed the tag from the
		// tagger.
		if wantTag != gotTag {
			t.Errorf("Pather got unexpected tag %v, want %v", gotTag, wantTag)
		}
		return want
	}

	got, err := lib.PathAt(0)
	if err != nil {
		t.Fatalf("lib.PathAt(0) = _, %v; want _, nil", err)
	}

	if got != want {
		t.Errorf("lib.PathAt(0) = %q, want %q", got, want)
	}
}
