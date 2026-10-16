package render

import (
	"strconv"
	"strings"

	"github.com/LZD121/x-terminal/internal/domain"
)

// CoverStyle names a display style a cover wears under immersive mode: the value the config's
// `display.cover` carries and the shell selects a Grammar by. The zero value is the plain reader
// and renders the plain grammar, so a caller that never set a style gets the reader and not a
// half-filled cover.
type CoverStyle string

// CoverAlias is what a cover calls one account. Every field is computed from the account's own
// fingerprint (CoverAliasOf) and none of them is read from the account itself, which is what makes
// hiding the real identity and keeping the alias stable across processes the same property.
type CoverAlias struct {
	// Handle is the machine name a log style prints in its identity field: two dictionary words
	// and a two-digit discriminator, never a hash, never an id.
	Handle string
	// Name is the person a git-style author line names.
	Name string
	// Login is Name lowercased as `first.last`, the shape a tool's owner column holds.
	Login string
	// Email is Login with the discriminator on it, at the cover's own domain.
	Email string
}

// CoverAliasOf is the cover's identity function: one account, aliased. It is pure — no I/O, no
// clock, no randomness, no map iteration — deterministic, and total: an author with no id and no
// handle still renders a fixed alias, so no path can render an empty identity column.
//
// The style is part of the signature because every caller that draws an identity knows the style
// it draws it under, and it is deliberately not read: all four projections are computed for every
// style from the two digests, and the style decides only which slot a grammar's head templates read
// — the machine name for a log or a build tail, the person's name and address for a git author
// line, the login for a table's owner column. That is what makes the two namespaces one value each:
// switching styles changes the spelling of an identity and nothing else, and a test asserts it.
func CoverAliasOf(style CoverStyle, u domain.User) CoverAlias {
	h1 := coverDigest(coverPurposeSlug, coverKey(u))
	h2 := coverDigest(coverPurposeName, coverKey(u))
	discriminator := twoDigits(int((h1 >> 16) % 100))
	name := coverFirst[h2%48] + " " + coverSurnames[(h2>>8)%64]
	login := strings.ToLower(coverFirst[h2%48] + "." + coverSurnames[(h2>>8)%64])
	return CoverAlias{
		Handle: coverRoles[h1%24] + "-" + coverNouns[(h1>>8)%64] + "-" + discriminator,
		Name:   name,
		Login:  login,
		Email:  login + discriminator + coverMailDomain,
	}
}

// coverKey is the one input rule of the fingerprint: the account's id when it has one, its handle
// lowercased when it does not, and the anonymous key when it has neither. Lowercasing is not
// cosmetic — the upstream hands the same account back with the handle spelled as it was typed —
// and the anonymous key is what makes an author nobody named still render one fixed alias.
func coverKey(u domain.User) string {
	if u.ID != "" {
		return u.ID
	}
	if handle := strings.ToLower(strings.TrimPrefix(u.ScreenName, "@")); handle != "" {
		return handle
	}
	return coverAnonymousKey
}

// coverDigest is the cover's one hash: FNV-1a 64-bit over the version segment, the field's purpose
// and the key, in that order and separated so no two fields can collide.
//
// FNV-1a is spelled out here, and frozen, for three reasons that are all the same reason: it is
// not seeded (hash/maphash is, per process, which would break the alias's stability invisibly), it
// iterates nothing (a map's order is not a fact about an account), and it reads nothing (no clock,
// no current list, no list length, no counter). The version segment is part of the input so that a
// later change of the word lists changes every alias deliberately, in one commit, with the stored
// frames as the diff.
func coverDigest(purpose, key string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	hash := uint64(offset64)
	for _, part := range [...]string{coverKeyVersion, purpose, "\x00", key} {
		for i := range len(part) {
			hash ^= uint64(part[i])
			hash *= prime64
		}
	}
	return hash
}

// twoDigits renders n as exactly two ASCII digits, for the discriminator that makes two accounts
// sharing a word pair tell each other apart.
func twoDigits(n int) string {
	return string([]byte{byte('0' + n/10), byte('0' + n%10)})
}

// The fingerprint's frozen inputs. The purposes are the fields a cover names a post by: the same
// post keeps the same token in every style that renders it, because the purpose is the field's and
// not the style's.
const (
	// coverKeyVersion is the fingerprint's version segment: a change of the word lists bumps it.
	coverKeyVersion = "xt-cover/v1/"
	// coverPurposeSlug is the purpose of the machine name's two digests.
	coverPurposeSlug = "slug"
	// coverPurposeName is the purpose of the person's two digests.
	coverPurposeName = "name"
	// coverPurposeSeq is the purpose of a service log's sequence number.
	coverPurposeSeq = "seq"
	// coverPurposeSHA is the purpose of git's abbreviated commit hash.
	coverPurposeSHA = "sha"
	// coverPurposeReq is the purpose of the number a build log and a request queue both print as
	// their record's reference. One purpose for both, because they are one field: the phase's own
	// renderings carry `build=4655` and `REQ-4655` for one post, and a post that changed its number
	// when the style changed would not be a record any more.
	coverPurposeReq = "req"
	// coverAnonymousKey is the key of an account with neither an id nor a handle: a fixed alias is
	// still an alias, and an empty identity column is not.
	coverAnonymousKey = "\x00anonymous"
	// coverMailDomain is the domain the covered address uses. It is a name that cannot be resolved
	// to anything of the user's: no company, no provider, no person.
	coverMailDomain = "@corp.internal"
)

// The word lists, compiled in and frozen in this order. They are the implementer's discretion as
// words and not as a rule: the indices, the list sizes and the frozen order are what the stored
// frames pin, and substituting a word recomputes every alias and every stored frame in the same
// commit. Every list is ASCII, no word holds a space, and the longest slug (role + `-` + noun + `-`
// + two digits) is 18 cells, which is why a cover's identity field is 20.
//
// The arrays are fixed-length on purpose: a list that lost a word would not compile, and no map is
// involved, because a map's iteration order is not a function of anything.
var (
	coverRoles = [24]string{
		"svc", "worker", "agent", "daemon", "cron", "runner", "relay", "probe",
		"drain", "batch", "broker", "spool", "tracer", "sentry", "scout", "hook",
		"tap", "pump", "sink", "shard", "guard", "mirror", "stream", "index",
	}
	coverNouns = [64]string{
		"deploy", "ingest", "ledger", "quota", "metrics", "billing", "report", "export",
		"search", "auth", "notify", "router", "storage", "webhook", "parser", "index",
		"media", "mailer", "pricing", "catalog", "invoice", "shipping", "payroll", "session",
		"archive", "stream", "filter", "digest", "snapshot", "backup", "restore", "migrate",
		"compact", "vacuum", "replay", "dedupe", "throttle", "backoff", "retry", "timeout",
		"lease", "token", "secret", "policy", "audit", "alert", "settle", "allocate",
		"reserve", "commit", "rollback", "dispatch", "fanout", "merge", "encode", "decode",
		"compress", "encrypt", "sign", "verify", "quorum", "cursor", "cache", "buffer",
	}
	coverFirst = [48]string{
		"Dana", "Mira", "Jonas", "Priya", "Tomas", "Elena", "Ravi", "Nadia",
		"Oscar", "Yuki", "Ines", "Marek", "Sofia", "Kwame", "Lena", "Arjun",
		"Clara", "Diego", "Hana", "Felix", "Amara", "Nils", "Rosa", "Tariq",
		"Iris", "Pavel", "Mei", "Owen", "Zara", "Bruno", "Anya", "Simon",
		"Leila", "Mateo", "Greta", "Idris", "Nora", "Viktor", "Salma", "Hugo",
		"Elif", "Rafael", "Astrid", "Kofi", "Vera", "Linus", "Camille", "Rune",
	}
	coverSurnames = [64]string{
		"Whitfield", "Okafor", "Lindqvist", "Marchetti", "Bhatt", "Novak", "Ferreira", "Tanaka",
		"Haddad", "Petrov", "Oyelaran", "Kowalski", "Duarte", "Nakamura", "Bergstrom", "Silva",
		"Rahman", "Vogel", "Costa", "Iyer", "Moreau", "Kuznetsov", "Nwosu", "Halvorsen",
		"Reyes", "Bianchi", "Sorensen", "Adeyemi", "Fontaine", "Krishnan", "Larsen", "Mensah",
		"Ricci", "Sato", "Varga", "Yilmaz", "Delacroix", "Osei", "Pereira", "Zhang",
		"Andersen", "Bogdan", "Cruz", "Eriksen", "Fischer", "Gupta", "Hoffmann", "Ivanov",
		"Jansen", "Karlsson", "Lindgren", "Mancini", "Nasser", "Olsen", "Popa", "Quintero",
		"Ravel", "Sokolov", "Tremblay", "Ueda", "Vasquez", "Weber", "Ximenes", "Yamada",
	}
)

// coverReference renders an integer token the way a log prints one, and is here rather than in
// cover.go because it is the same shape of function as the projections above: a frozen input, one
// number out.
func coverReference(base, span int, purpose, key string) string {
	return strconv.Itoa(base + int(coverDigest(purpose, key)%uint64(span)))
}
