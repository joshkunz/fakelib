package library

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"strings"

	"github.com/bogem/id3v2"
)

// Song is the type of a song in the library. It can be generated via Library.SongAt().
type Song struct {
	tag  []byte
	data []byte
}

// Size is the size in bytes of this song.
func (s Song) Size() int64 {
	return int64(len(s.tag) + len(s.data))
}

// Read reads bytes from this song into the buffer `buf` starting at byte `off`
// in the song. All data is read from memory, so this operation cannot fail.
func (s Song) Read(buf []byte, off int64) {
	// Nothing to read here.
	if off >= s.Size() {
		return
	}

	if off < int64(len(s.tag)) {
		read := copy(buf, s.tag[off:])
		buf = buf[read:]
		// If off < len(e.tag), the we've read all we can from
		// the tag, and we should re-start at the beginning of
		// the song.
		off = 0
	} else {
		// Otherwise, we need to just read from the song, and we
		// should exclude the tag part from the offset.
		off -= int64(len(s.tag))
	}
	copy(buf, s.data[off:])
}

// Library represents a fake library of songs. A single "golden" MP3 is
// used as the basis for every track in the library, and song metadata is
// generated on a per-track basis. A new library can be created with `New`.
// The number of tracks, and the structure of the library can be controlled
// via member variables.
//
// Songs in the library are always generated in the form:
//    <artist>/<album>/<track>.mp3
// Where each component is some number of characters from A-Z.
// Artists/Albums/Tracks are named in-order, starting at 0. So track 0 is
//    A/A/A.mp3
// Track 1 is:
//    A/A/B.mp3
// etc.
// Track metadata represents what is shown in the path, except that the track
// title is the concatenation of <artist>, <album>, <track> with "-" as a
// separator.
//
// When MinPathLength > 3, path components are duplicated to extend the length
// of the path, while maintaining uniqueness. E.g., when MinPathLength = 4,
// Track 0 is:
//    AA/AA/AA.mp3
//
// When all letters have been exhausted in a category, the name is extended
// following a "spreadsheet" schema: A, B, ..., Z, AA, AB, ..., ZZ, AAA, ...
// When MinPathLength > 3, the repeated name is extended. So when
// MinPathLength = 4, "AB" becomes "ABAB".
type Library struct {
	// Total number of tracks in the fake library.
	Tracks int

	TracksPerAlbum  int
	AlbumsPerArtist int
	// Number of artists is derived from #of tracks, the track/album, and
	// album/artist ratios.

	// The minimum length of a path. Path elements are repeated to extend this
	// value. Must be >= 3 or the result is undefined.
	MinPathLength int

	// golden is the "golden" track data for this
	// Library. Does not include id3v2 header.
	golden []byte
}

var letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func letterName(i int) string {
	// Increment by one, so that our loop below can
	// be in terms of i > 0.
	i++
	var name []byte
	for ; i > 0; i /= len(letters) {
		letter := i % len(letters)
		name = append(name, letters[letter-1])
	}
	return string(name)
}

func (l *Library) name(i int) string {
	minLength := l.MinPathLength
	if minLength == 0 {
		// Special case to make the zero-value useful. Assume 3.
		minLength = 3
	}
	// Divide by 3 because our paths have 3 components.
	extension := minLength / 3
	if minLength%3 != 0 {
		extension++
	}
	return strings.Repeat(letterName(i), extension)
}

func (l *Library) attrsAt(idx int) (artist, album, name string, track int) {
	if idx < 0 || idx > (l.Tracks-1) {
		panic("index should be sanitized before attrsAt is called")
	}

	artistIdx := idx / (l.TracksPerAlbum * l.AlbumsPerArtist)
	albumIdx := (idx / l.TracksPerAlbum) % l.AlbumsPerArtist
	trackIdx := idx % l.TracksPerAlbum

	artist = l.name(artistIdx)
	album = l.name(albumIdx)
	name = l.name(trackIdx)

	// Tracks on the album are numbered starting at 1, so trackIdx+1
	return artist, album, name, trackIdx + 1
}

// PathAt returns the path to the idx-th song in the library.
func (l *Library) PathAt(idx int) (string, error) {
	if idx < 0 || idx > (l.Tracks-1) {
		return "", fmt.Errorf("index %d out of range [0, %d)", idx, l.Tracks)
	}

	artist, album, track, _ := l.attrsAt(idx)
	return path.Join(artist, album, track) + ".mp3", nil
}

// SongAt returns the song at the idx-th spot in the library.
func (l *Library) SongAt(idx int) (Song, error) {
	if idx < 0 || idx > (l.Tracks-1) {
		return Song{}, fmt.Errorf("index %d out of range [0, %d)", idx, l.Tracks)
	}

	artist, album, name, track := l.attrsAt(idx)

	t := id3v2.NewEmptyTag()
	t.SetArtist(artist)
	t.SetAlbum(album)
	t.SetTitle(fmt.Sprintf("%s - %s - %s", artist, album, name))
	t.AddTextFrame(
		t.CommonID("Track number/Position in set"),
		id3v2.EncodingUTF8,
		strconv.Itoa(track),
	)

	var buf bytes.Buffer
	if _, err := t.WriteTo(&buf); err != nil {
		log.Fatalf("error writing id3v2 header to buffer: %v", err)
	}

	return Song{tag: buf.Bytes(), data: l.golden}, nil
}

// New returns a new Library that uses Golden data
func New(golden io.ReadSeeker) (*Library, error) {
	header, err := id3v2.ParseReader(golden, id3v2.Options{Parse: true})
	if err != nil {
		return nil, fmt.Errorf("failed to parse id3v2 header: %v", err)
	}

	// Re-seek in-case the id3v2 library read more than the header.
	if _, err := golden.Seek(int64(header.Size()), io.SeekStart); err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(golden)
	if err != nil {
		return nil, err
	}

	return &Library{
		Tracks:          1000,
		TracksPerAlbum:  10,
		AlbumsPerArtist: 3,
		MinPathLength:   3,
		golden:          data,
	}, nil
}
