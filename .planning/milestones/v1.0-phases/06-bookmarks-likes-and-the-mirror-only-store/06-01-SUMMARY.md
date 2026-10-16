---
phase: 06-bookmarks-likes-and-the-mirror-only-store
plan: 01
type: summary
status: complete
requirements: [SAVE-05, SEC-02]
verified: |
  gofmt -l internal cmd                         empty
  CGO_ENABLED=0 go build ./...                  ok
  go vet ./internal/store/... ./internal/domain/...  ok
  go test ./... -count=1                        20 packages ok (store+domain 31 tests)
  go test ./internal/store/... -count=1         inside unshare -rn (lo up)  ok
  grep -c 'modernc.org/sqlite' go.mod           1
  grep -c 'user_version' internal/store/*.go    0
  grep -c 'DELETE' internal/store/*.go          0
  go doc ./internal/store                       Open/Close/Path/Page/Status/Upsert/MarkAbsent/RecordRun/SetOwner
  sqlite3 saved.db '.tables'                    meta saved_items sync_runs
  sqlite3 … schema_version                      1
  planted internal/store import in xsource/home.go → TestBoundaryImportScan FAIL; revert → PASS
unverified: |
  Begin/Step/Abandon SyncRun implementation — Mirror.Begin is declared on the
  domain port; this plan freezes the sealed write API and reader, not the walk
  driver (later plans).
---
# 06-01 SUMMARY — mirror-only store spine (SAVE-05, SEC-02)

## 1. Domain ports and values

`internal/domain/saved.go` — `SavedKind` (Bookmark/Like), `ListKind` +
`HomeList`/`SavedList` round trips, `SavedItem`, `RunID`, `StopReason`,
`SyncProgress`/`SyncResult`, `MirrorStatus`, `Capability`, and the `Mirror` /
`SyncRun` ports. Totality tables in `saved_test.go`.

## 2. Store file, WAL read-back, sealed write API

`internal/store/schema.go` — `Store{path,w,r}`, `Open` (writer MaxOpenConns=1,
reader `query_only`, `migrate`, `PRAGMA journal_mode` must read `wal`),
`schemaV1` three tables, `declaredTables` / `declaredMetaKeys`.
`migrate.go` — ordered `[]migration`, version bump in the same transaction as
the DDL, `ErrSchemaNewer`, version from `meta` (no `user_version`).
`read.go` — `Page` (20-row keyset, present=1) and `Status` on the reader pool.
`sync.go` — `Upsert`, `MarkAbsent` (never DELETE), `RecordRun` (last-success
only on successful stops), `SetOwner` (`ErrOwnerConflict`), private `snapshot`
with no engagement counts.

Decision: `migrate` is unexported so the exported surface stays the nine-name
allowlist; Open is the only file-open entry.

## 3. Proofs that can fail

- Schema allowlist vs `sqlite_master` / `PRAGMA table_info`; extra table and
  extra column arms fail the comparator; ≥3 tables / ≥10 columns anti-vacuity.
- Torn migration (DDL that fails mid-way) leaves version 1 and rows intact;
  newer schema refused; upgrade keeps rows.
- Boundary AST scan over xsource/screen/render/tui: no `internal/store` import;
  planted fixture fails; `database/sql` in store and bubbletea in tui as
  positive controls (31 files / 104 imports logged).
- Fake adversarial timeline session through `screen` reducers leaves data dir =
  prior files + `saved.db`(-wal/-shm) and `saved_items` empty.

## 4. Dependency

`modernc.org/sqlite` v1.60.1 only; `CGO_ENABLED=0` build preserved.
