# go-hibp-sync

`go-hibp-sync` provides functionality to keep a local copy of the *HIBP leaked password database* in sync with the upstream version at [https://haveibeenpowned.com](https://haveibeenpowned.com).
In addition to syncing the "database", the library allows exporting it into a single list — the former distribution format of the database — and querying it for a given *k-proximity range*. 

This local copy consists of one file per range/prefix, grouped into `256` directories (first `2` of `5` prefix characters).
As an uncompressed copy of the database would currently require around `~40 GiB` of disk space, a moderate level of `zstd` compression is applied with the result of cutting down storage consumption by `50%`. 
This compression can be disabled if the little computational overhead caused outweighs the advantage of requiring only half the space.

To avoid unnecessary network transfers and to also speed up things, `go-hibp-sync` additionally keeps the `etag` returned by the upstream CDN.
Subsequent requests contain it and should allow for more frequent syncs, not necessarily resulting in full re-downloads.
Of course, this can be disabled too.

The library supports to continue from where it left off, the `sync` command mentioned below demonstrates this.


## API

The API is really simple; one type, holding three methods, is exported (and additionally, typed configuration options):

```go
New(options ...CommonOption) *HIBP
HIBP#Sync(options ...SyncOption) error // Syncs the local copy with the upstream database
HIBP#Export(w io.Writer, options ...ExportOption) error // Writes a continuous, decompressed and "free-of-etags" stream to the given io.Writer with the lines being prefix by the k-proximity range
HIBP#.Query("ABCDE") (io.ReadClose, error) // Returns the k-proximity API result as the upstream API would (without the k-proximity range as prefix)
```

All of them operate on disk but, depending on the medium, should provide access times that are probably good enough for all scenarios.
A memory-based `tmpfs` will speed things up when necessary.

**Attention:**
The [official API](https://haveibeenpwned.com/API/v3#PwnedPasswords) states the following regarding the format: 

> Each password is stored as both a SHA-1 and an NTLM hash of a UTF-8 encoded password. 
> The downloadable source data delimits the hash and the password count with a colon (:) and each line with a CRLF.

The crucial part being that lines are ended with `\r\n`.
In order to be compatible with the upstream API this library sticks to this...


## CLI

There are two basic CLI commands, `sync` and `export` that can be used for manual tasks and serve as minimal examples on how to use the library.
They are basic but should play well with other tooling.
`sync` will track the progress and is able to continue from where it left of last.

Run them with:

```bash
go run github.com/exaring/go-hibp-sync/cmd/sync
# and
go run github.com/exaring/go-hibp-sync/cmd/export
```