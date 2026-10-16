package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LZD121/x-terminal/internal/audit"
)

func TestAuditCleanRun(t *testing.T) {
	prev := auditRun
	auditRun = func(root string) ([]audit.Report, error) {
		return []audit.Report{
			{Check: "readonly", Promise: "no write", Files: 3},
			{Check: "transport", Promise: "no mutation", Files: 3},
		}, nil
	}
	t.Cleanup(func() { auditRun = prev })

	// repoRoot needs a real tree; run from module root.
	root := moduleRoot(t)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	code, stdout, stderr := run(t, Audit)
	if code != ExitOK {
		t.Fatalf("audit exited %d (%s), want %d", code, stderr, ExitOK)
	}
	if stderr != "" {
		t.Errorf("clean run wrote stderr: %q", stderr)
	}
	if !strings.Contains(stdout, "readonly") || !strings.Contains(stdout, "3 files") {
		t.Errorf("report missing check line: %q", stdout)
	}
	if !strings.Contains(stdout, "0 findings") {
		t.Errorf("report missing closing line: %q", stdout)
	}
}

func TestAuditStubbedFinding(t *testing.T) {
	prev := auditRun
	auditRun = func(root string) ([]audit.Report, error) {
		return []audit.Report{{
			Check: "readonly", Promise: "no write", Files: 2,
			Findings: []audit.Finding{{
				Check: "readonly", Path: "internal/xsource/x.go", Line: 12, Detail: "write operation name CreateTweet",
			}},
		}}, nil
	}
	t.Cleanup(func() { auditRun = prev })

	root := moduleRoot(t)
	old, _ := os.Getwd()
	_ = os.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(old) })

	code, stdout, stderr := run(t, Audit)
	if code != ExitError {
		t.Fatalf("audit with finding exited %d, want %d", code, ExitError)
	}
	if !strings.Contains(stdout, "internal/xsource/x.go:12:") {
		t.Errorf("finding not printed as file:line: %q", stdout)
	}
	if stderr != "" {
		t.Errorf("findings went to stderr: %q", stderr)
	}
}

func TestAuditJSON(t *testing.T) {
	prev := auditRun
	auditRun = func(root string) ([]audit.Report, error) {
		return []audit.Report{{
			Check: "timers", Promise: "no timers", Files: 4,
			Findings: []audit.Finding{{Path: "internal/x.go", Line: 3, Detail: "banned timer call time.NewTicker"}},
		}}, nil
	}
	t.Cleanup(func() { auditRun = prev })

	root := moduleRoot(t)
	old, _ := os.Getwd()
	_ = os.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(old) })

	code, stdout, stderr := run(t, Audit, "--json")
	if code != ExitError {
		t.Fatalf("json audit exited %d (%s)", code, stderr)
	}
	var got struct {
		Checks []struct {
			Name     string `json:"name"`
			Files    int    `json:"files"`
			Findings []struct {
				Path   string `json:"path"`
				Line   int    `json:"line"`
				Detail string `json:"detail"`
			} `json:"findings"`
		} `json:"checks"`
		Findings int `json:"findings"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout)
	}
	if got.Findings != 1 || len(got.Checks) != 1 || got.Checks[0].Findings[0].Line != 3 {
		t.Fatalf("unexpected json: %+v", got)
	}
}

func TestAuditUnloadableTree(t *testing.T) {
	prev := auditRun
	auditRun = func(root string) ([]audit.Report, error) {
		return nil, fmt.Errorf("open %s: permission denied", root)
	}
	t.Cleanup(func() { auditRun = prev })

	root := moduleRoot(t)
	old, _ := os.Getwd()
	_ = os.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(old) })

	code, stdout, stderr := run(t, Audit)
	if code != ExitError {
		t.Fatalf("unloadable tree exited %d", code)
	}
	if stdout != "" {
		t.Errorf("unloadable tree wrote stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "permission denied") {
		t.Errorf("stderr missing reason: %q", stderr)
	}
}

func TestAuditListedInUsage(t *testing.T) {
	var names []string
	for _, c := range commands() {
		names = append(names, c.Name)
	}
	if !strings.Contains(strings.Join(names, ","), "audit") {
		t.Fatalf("commands missing audit: %v", names)
	}
}

// TestAuditJSONCarriesThePromiseWhole: the declared destination set and the
// call sites that reached it have to survive into the JSON twin, or the
// machine-readable report would state less than the text one.
func TestAuditJSONCarriesThePromiseWhole(t *testing.T) {
	long := "every write resolves to one of the 6 named destinations: session, pins, notice, saved (the running tool); registry, spike-recording (a development tool) [session@internal/config/config.go:461, pins@internal/xweb/volatile/pins.go:77, registry@internal/xweb/volatile/registry.go:399, registry@internal/xweb/volatile/registry.go:410, spike-recording@cmd/spike/main.go:176]"
	prev := auditRun
	auditRun = func(string) ([]audit.Report, error) {
		return []audit.Report{{Check: "persistence", Promise: long, Files: 98}}, nil
	}
	t.Cleanup(func() { auditRun = prev })

	root := moduleRoot(t)
	old, _ := os.Getwd()
	_ = os.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(old) })

	code, stdout, stderr := run(t, Audit, "--json")
	if code != ExitOK {
		t.Fatalf("json audit exited %d (%s)", code, stderr)
	}
	var got struct {
		Checks []struct {
			Promise string `json:"promise"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout)
	}
	if len(got.Checks) != 1 || got.Checks[0].Promise != long {
		t.Fatalf("promise not carried whole: %q", got.Checks[0].Promise)
	}
	if !strings.Contains(got.Checks[0].Promise, "spike-recording@cmd/spike/main.go:176") {
		t.Fatalf("promise lost the call-site evidence: %q", got.Checks[0].Promise)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// package is internal/cli
	root := filepath.Clean(filepath.Join(wd, "../.."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(err)
	}
	return root
}
