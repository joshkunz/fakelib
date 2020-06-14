package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"fakelib/library"
)

var (
	librarySize     = flag.Int("library_size", 1000, "Number of songs to include in the library")
	minPathLength   = flag.Int("min_path_length", 3, "The minimum number of non-separator bytes in the generated paths")
	tracksPerAlbum  = flag.Int("tracks_per_album", 10, "Max number of tracks in each album")
	albumsPerArtist = flag.Int("albums_per_artist", 3, "Max number of albums for each artist")
)

type songInode struct {
	fs.Inode

	song library.Song
}

var _ fs.NodeOpener = (*songInode)(nil)
var _ fs.NodeReader = (*songInode)(nil)
var _ fs.NodeGetattrer = (*songInode)(nil)

func (s *songInode) Open(context.Context, uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, fs.OK
}

func (s *songInode) Read(_ context.Context, _ fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	s.song.Read(dest, off)
	return fuse.ReadResultData(dest), fs.OK
}

func (s *songInode) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Size = uint64(s.song.Size())
	return fs.OK
}

type root struct {
	fs.Inode

	l *library.Library
}

var _ fs.NodeOnAdder = (*root)(nil)

func (r *root) OnAdd(ctx context.Context) {
	for i := 0; i < r.l.Tracks; i++ {
		location, err := r.l.PathAt(i)
		if err != nil {
			log.Fatalf("failed to get path at idx %d: %v", i, err)
		}
		song, err := r.l.SongAt(i)
		if err != nil {
			log.Fatalf("failed to get song at idx %d: %v", i, err)
		}
		dir, fname := path.Split(location)

		wd := &r.Inode
		for _, component := range strings.Split(dir, "/") {
			if component == "" {
				continue
			}

			cur := wd.GetChild(component)
			if cur == nil {
				cur = wd.NewPersistentInode(ctx, &fs.Inode{}, fs.StableAttr{Mode: fuse.S_IFDIR})
				wd.AddChild(component, cur, true)
			}

			wd = cur
		}

		node := wd.NewPersistentInode(ctx, &songInode{song: song}, fs.StableAttr{})
		wd.AddChild(fname, node, true)
	}
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		log.Fatalf("usage: %s golden.mp3 mount/", os.Args[0])
	}

	goldenPath, mountDir := flag.Arg(0), flag.Arg(1)

	golden, err := os.Open(goldenPath)
	if err != nil {
		log.Fatalf("failed to open golden file %q: %v", goldenPath, err)
	}

	lib, err := library.New(golden)
	if err != nil {
		log.Fatalf("failed to load golden file %q: %v", goldenPath, err)
	}
	lib.Tracks = *librarySize
	lib.TracksPerAlbum = *tracksPerAlbum
	lib.AlbumsPerArtist = *albumsPerArtist
	lib.MinPathLength = *minPathLength

	// No need for the file anymore, just close it to drop the handle.
	golden.Close()

	if *minPathLength < 3 {
		log.Fatalf("--min_path_length must be at least 3")
	}

	if _, err := os.Stat(mountDir); os.IsNotExist(err) {
		os.Mkdir(mountDir, 0755)
	} else if err != nil {
		log.Fatalf("failed to stat %q: %v", mountDir, err)
	}

	server, err := fs.Mount(mountDir, &root{l: lib}, nil)
	if err != nil {
		log.Fatal(err)
	}
	server.Serve()
	server.Wait()
}
