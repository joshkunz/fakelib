/*
Package filesystem provides a FUSE filesystem that can be used to mount
a fake library to a particular filesystem path.

Typical Usage:

    lib, err := library.New(...)
    if err != nil {
        ...
    }

    server, err := filesystem.Mount(lib, dir, nil)
    if err != nil {
        ...
    }
    server.Serve()
*/
package filesystem

import (
	"context"
	"log"
	"path"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/joshkunz/fakelib/library"
)

type song struct {
	fs.Inode

	song library.Song
}

var _ fs.NodeOpener = (*song)(nil)
var _ fs.NodeReader = (*song)(nil)
var _ fs.NodeGetattrer = (*song)(nil)

func (s *song) Open(context.Context, uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, fs.OK
}

func (s *song) Read(_ context.Context, _ fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	s.song.Read(dest, off)
	return fuse.ReadResultData(dest), fs.OK
}

func (s *song) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
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
		lSong, err := r.l.SongAt(i)
		if err != nil {
			log.Fatalf("failed to get song at idx %d: %v", i, err)
		}
		dir, fname := path.Split(location)

		wd := &r.Inode
		for _, component := range strings.Split(dir, "/") {
			if component == "" {
				// `dir` likely has a trailing `/` which yields an empty path
				// component on split, so ignore that component.
				continue
			}

			cur := wd.GetChild(component)
			if cur == nil {
				cur = wd.NewPersistentInode(ctx, &fs.Inode{}, fs.StableAttr{Mode: fuse.S_IFDIR})
				wd.AddChild(component, cur, true)
			}

			wd = cur
		}

		node := wd.NewPersistentInode(ctx, &song{song: lSong}, fs.StableAttr{})
		wd.AddChild(fname, node, true)
	}
}

// Mount mounts the given library into `dir`. `options` can be used to supply
// additional FUSE mount options. If the default options are OK, then `nil`
// can safely be provided for `options`.
func Mount(lib *library.Library, dir string, options *fs.Options) (*fuse.Server, error) {
	return fs.Mount(dir, &root{l: lib}, options)
}
