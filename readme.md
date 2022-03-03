# fakelib

[![API reference](https://img.shields.io/badge/go.pkg.dev-reference-5272B4)](
https://pkg.go.dev/github.com/joshkunz/fakelib?tab=doc)
[![Test](https://github.com/joshkunz/fakelib/actions/workflows/test.yaml/badge.svg)](
https://github.com/joshkunz/fakelib/actions/workflows/test.yaml)
[![LICENSE](
https://img.shields.io/github/license/joshkunz/ashuffle?color=informational)](
LICENSE)

`fakelib` is a tool for generating massive music libraries for testing
purposes. It was originally created as part of the [ashuffle project](
https://github.com/joshkunz/ashuffle) to test the performance of `ashuffle`.

Install:

```
go install github.com/joshkunz/fakelib@latest
```

## Why?

`ashuffle`'s users often had libraries in the 10s to 100s of thousands of songs.
Since `ashuffle` had to track a user's library, there were performance issues
scaling to these massive numbers of tracks. Since `ashuffle` did not interact
with the libraries directly (instead it queried `MPD`), the libraries used
during testing had to be real, valid, MP3s with correct metadata. 

The first approach at generating large libraries was to generate a "golden"
MP3, copy it several thousand times, and update the metadata accordingly. This
worked, but was painful to scale past the low 10s of thousands of songs.
Additionally this approach had substantial drawbacks. First, it took almost an
hour to generate, since it was limited to disk throughput. It also consumed
a large amount of space (nearly 500MB for 20k tracks). This made it expensive
and slow to ship around to test infrastructure. Worst of all, even with those
drawbacks, the libraries it generated were still much smaller than some users
libraries.

`fakelib` is an attempt to generate much larger libraries. `fakelib` can easily
generate a fake library with a million tracks in seconds. `fakelib` is similar
in concept to copying an MP3 several thousand times and updating the metadata,
however, it's much more efficient. Instead of actually doing the copies
`fakelib` implements a FUSE filesystem. When mounted this filesystem appears
as a large music library. When a program tries to read one of the MP3s in the
library, `fakelib` generates tag information and responds with the bytes of
the generated tag. Once the tag has been read, later reads in the file go
directly to a "golden" MP3's audio data. This means that the library is
functionally a large library of tracks with unique metadata, and exactly the
same audio data. The "golden" MP3 is never copied.

## Running

First you'll need a "golden" MP3 which `fakelib` will use as the audio data
for all fake tracks. Any MP3 should work, but you can also generate a short
empty MP3 using `ffmpeg`. This is ideal for testing since it takes up very
little space:

```
$ SECONDS=5
$ ffmpeg -f lavfi -i anullsrc=r=44100:cl=mono -t $SECONDS -q:a 9 -acodec libmp3lame gold.mp3
```

Once you have a golden MP3, you can run `fakelib` to mount the fake library
file system, and then use it as normal:

```
$ mkdir test
$ fakelib gold.mp3 ./test/
```

## As a Library

`fakelib` can also be used as a library. See the documentation for details.
