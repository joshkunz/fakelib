package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/joshkunz/fakelib/filesystem"
	"github.com/joshkunz/fakelib/library"
)

var (
	librarySize     = flag.Int("library_size", 1000, "Number of songs to include in the library")
	minPathLength   = flag.Int("min_path_length", 3, "The minimum number of non-separator bytes in the generated paths")
	tracksPerAlbum  = flag.Int("tracks_per_album", 10, "Max number of tracks in each album")
	albumsPerArtist = flag.Int("albums_per_artist", 3, "Max number of albums for each artist")
)

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatalf("usage: %s golden.mp3 mount/", os.Args[0])
	}

	if *minPathLength < 3 {
		log.Fatalf("--min_path_length must be at least 3")
	}

	var goldenPath, mountDir string
	if len(flag.Args()) < 2 {
		mountDir = flag.Arg(0)
	} else {
		goldenPath, mountDir = flag.Arg(0), flag.Arg(1)
	}

	var golden io.ReadSeeker
	if goldenPath != "" {
		var err error
		golden, err = os.Open(goldenPath)
		if err != nil {
			log.Fatalf("failed to open golden file %q: %v", goldenPath, err)
		}
		defer golden.(*os.File).Close()
	} else {
		golden = library.EmbeddedGoldMP3()
	}

	lib, err := library.New(golden)
	if err != nil {
		log.Fatalf("failed to load golden file %q: %v", goldenPath, err)
	}
	lib.Tracks = *librarySize
	lib.Tagger = library.RepeatedLetters{
		TracksPerAlbum:     *tracksPerAlbum,
		AlbumsPerArtist:    *albumsPerArtist,
		MinComponentLength: *minPathLength / 3,
	}.Tag

	if _, err := os.Stat(mountDir); os.IsNotExist(err) {
		os.Mkdir(mountDir, 0755)
	} else if err != nil {
		log.Fatalf("failed to stat %q: %v", mountDir, err)
	}

	server, err := filesystem.Mount(lib, mountDir, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("filesystem mounted at %q\n", mountDir)

	// Wait for our process to be interrupted.
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	if err := server.Unmount(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("filesystem unmounted from %q\n", mountDir)
}
