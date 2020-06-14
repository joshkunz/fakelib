package library

import (
	"bytes"
	"log"
	"testing"

	"github.com/bogem/id3v2"
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
			Title:  "A - A - A",
			Track:  "1",
		},
	},
	{
		idx:          1,
		wantLocation: "A/A/B.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "A",
			Title:  "A - A - B",
			Track:  "2",
		},
	},
	{
		idx:          2,
		wantLocation: "A/A/C.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "A",
			Title:  "A - A - C",
			Track:  "3",
		},
	},
	{
		idx:          10,
		wantLocation: "A/B/A.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "B",
			Title:  "A - B - A",
			Track:  "1",
		},
	},
	{
		idx:          11,
		wantLocation: "A/B/B.mp3",
		wantInfo: trackInfo{
			Artist: "A",
			Album:  "B",
			Title:  "A - B - B",
			Track:  "2",
		},
	},
	{
		idx:          30,
		wantLocation: "B/A/A.mp3",
		wantInfo: trackInfo{
			Artist: "B",
			Album:  "A",
			Title:  "B - A - A",
			Track:  "1",
		},
	},
	{
		idx:          31,
		wantLocation: "B/A/B.mp3",
		wantInfo: trackInfo{
			Artist: "B",
			Album:  "A",
			Title:  "B - A - B",
			Track:  "2",
		},
	},
	{
		idx:          40,
		wantLocation: "B/B/A.mp3",
		wantInfo: trackInfo{
			Artist: "B",
			Album:  "B",
			Title:  "B - B - A",
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
			Title:  "AA - A - A",
			Track:  "1",
		},
	},
	{
		idx:          (26 * 3 * 10) + 1,
		wantLocation: "AA/A/B.mp3",
		wantInfo: trackInfo{
			Artist: "AA",
			Album:  "A",
			Title:  "AA - A - B",
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
