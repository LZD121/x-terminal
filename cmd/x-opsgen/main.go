// Command x-opsgen refreshes the operation registry from a real x.com web bundle.
//
// It is a development tool, not a verb of the product: `x-terminal` never runs it, and it is
// the only thing in this repository that *writes* registry.json. Run it from the repository
// root, because the file it rewrites by default is the one the binary embeds at build time —
// a refresh is only in effect after a rebuild, which is the point: the ids are reviewed in a
// diff rather than mutating under a running client.
//
//	x-opsgen --from-file <bundle.js> [--merge] [--registry <path>]
//	x-opsgen --live [--merge] [--registry <path>]
//
//	--from-file   mine the operations out of a downloaded bundle
//	--live        fetch the authenticated shell and its bundle through the configured proxy:
//	              this needs the session cookie, and it is the only mode that spends requests
//	--merge       write the result back; without it the run is a preview and changes nothing
//	--registry    the file to read and rewrite
//	              (default internal/xweb/volatile/registry.json)
//
// Nothing here can delete an operation: the merge adds and updates, and a refresh that meets
// a user's pin leaves it alone (see internal/xweb/volatile).
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/LZD121/x-terminal/internal/config"
	"github.com/LZD121/x-terminal/internal/opsgen"
	"github.com/LZD121/x-terminal/internal/session"
	"github.com/LZD121/x-terminal/internal/xweb"
	"github.com/LZD121/x-terminal/internal/xweb/volatile"
)

// defaultRegistry is where the embedded registry lives, relative to the repository root.
const defaultRegistry = "internal/xweb/volatile/registry.json"

const usage = `usage: x-opsgen (--from-file <bundle.js> | --live) [--merge] [--registry <path>]

  --from-file <path>  mine the operations out of a bundle already on disk
  --live              fetch the authenticated shell and its bundle through the proxy
  --merge             write the merged registry back (without it: a preview, nothing written)
  --registry <path>   the registry file to read and rewrite (default ` + defaultRegistry + `)
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var (
		fromFile string
		live     bool
		merge    bool
		registry = defaultRegistry
	)
	// A classic index loop rather than `range len(args)`: the path flags consume the argument
	// after them, so this loop moves the index itself.
	for i := 0; i < len(args); i++ {
		arg, value := splitFlag(args[i])
		switch arg {
		case "--live":
			live = true
		case "--merge":
			merge = true
		case "--help", "-h":
			fmt.Fprint(stdout, usage)
			return 0
		case "--from-file", "--registry":
			if value == "" {
				i++
				if i >= len(args) {
					return bad(stderr, "%s needs a path", arg)
				}
				value = args[i]
			}
			if arg == "--from-file" {
				fromFile = value
			} else {
				registry = value
			}
		default:
			return bad(stderr, "unknown argument %s", args[i])
		}
	}

	if (fromFile == "") == !live {
		return bad(stderr, "give exactly one of --from-file <bundle.js> and --live")
	}

	ctx := context.Background()
	records, source, err := gather(ctx, fromFile, live)
	if err != nil {
		fmt.Fprintf(stderr, "x-opsgen: %v\n", err)
		return 1
	}
	if live {
		fmt.Fprintf(stdout, "mined %d operations from %s (%d bytes, sha256 %s)\n", len(records), source.url, source.bytes, source.sha)
	}

	reg, err := volatile.ReadFile(registry)
	if err != nil {
		fmt.Fprintf(stderr, "x-opsgen: %v\n", err)
		return 1
	}
	result := reg.Merge(opsgen.Operations(records))
	fmt.Fprintf(stdout, "%s (%d operations in %s)\n", result, reg.Len(), registry)

	if !merge {
		fmt.Fprintf(stdout, "preview only: pass --merge to rewrite %s\n", registry)
		return 0
	}
	reg.RecordRefresh(source.at, source.name)
	if err := reg.Write(registry); err != nil {
		fmt.Fprintf(stderr, "x-opsgen: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s — rebuild for it to take effect\n", registry)
	return 0
}

// source is where a run's records came from, for the provenance the file records and the line
// this tool prints.
type source struct {
	name  string // the bundle URL, or the path the bundle was read from
	url   string // the bundle URL, empty for a local file
	sha   string
	bytes int
	at    time.Time
}

// gather produces the records one run merges: from a file on disk, or from the live site.
func gather(ctx context.Context, fromFile string, live bool) ([]opsgen.Record, source, error) {
	if !live {
		bundle, err := os.ReadFile(fromFile)
		if err != nil {
			return nil, source{}, fmt.Errorf("reading %s: %w", fromFile, err)
		}
		records, err := opsgen.Parse(bundle)
		if err != nil {
			return nil, source{}, fmt.Errorf("%s: %w", fromFile, err)
		}
		return records, source{name: fromFile, bytes: len(bundle), at: time.Now().UTC()}, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, source{}, err
	}
	for _, warning := range cfg.Warnings {
		fmt.Fprintf(os.Stderr, "x-opsgen: %s\n", warning)
	}
	sess, err := session.Load(cfg.SessionPath)
	if err != nil {
		return nil, source{}, err
	}
	client, err := xweb.NewClient(cfg)
	if err != nil {
		return nil, source{}, err
	}

	refresh, err := opsgen.Fetch(ctx, client, sess, opsgen.ShellURL)
	if err != nil {
		return nil, source{}, err
	}
	return refresh.Records, source{name: refresh.BundleURL, url: refresh.BundleURL, sha: refresh.SHA256, bytes: refresh.Bytes, at: refresh.At}, nil
}

// splitFlag splits --flag=value into its two halves; a flag written without a value comes back
// whole, with an empty value.
func splitFlag(arg string) (flag, value string) {
	if !strings.HasPrefix(arg, "--") {
		return arg, ""
	}
	if name, value, ok := strings.Cut(arg, "="); ok {
		return name, value
	}
	return arg, ""
}

// bad reports a bad invocation the way every other verb in this repository does: the mistake
// on stderr, the usage under it, and a non-zero exit code.
func bad(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "x-opsgen: "+format+"\n\n", args...)
	fmt.Fprint(stderr, usage)
	return 2
}
