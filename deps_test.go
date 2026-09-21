package version

import (
	"os/exec"
	"strings"
	"testing"
)

// TestRootPackageStaysDependencyFree is the guard for the reason fiberadapter
// and httpadapter exist. Both were moved out so that a binary which only wants
// build information -- overwhelmingly a CLI printing --version -- does not
// link a web server it never starts.
//
// Fiber was the first to go: it drags fasthttp behind it, dead weight and CVE
// surface a net/http service never asked for. net/http followed in v4, and it
// is the larger of the two: it alone is 124 of the 200 packages the v3 root
// cost every importer, and a little over half the binary.
//
// Nothing else notices if that regresses. Adding `import "net/http"` back to a
// root-package file -- for an http.StatusOK constant, say -- compiles, passes
// every other test, and quietly puts those 124 packages back into every
// importer's binary. This test is what fails instead.
//
// Test files are exempt by construction: `go list -deps .` reports the
// package's own import graph, not its tests', so the root's tests may use
// net/http/httptest freely.
func TestRootPackageStaysDependencyFree(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH; cannot inspect the import graph")
	}

	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps .: %v\n%s", err, out)
	}

	// net/http is matched exactly. Prefix-matching it would also catch
	// net/http/httptest, which is fine, but it would not catch the thing that
	// actually matters any better -- and "net/http" is what an importer sees.
	banned := map[string]string{
		"github.com/gofiber/":         "use the fiberadapter subpackage",
		"github.com/valyala/fasthttp": "use the fiberadapter subpackage",
		"net/http":                    "use the httpadapter subpackage",
		"golang.org/x/net/http2":      "use the httpadapter subpackage",
	}

	var linked []string
	for _, pkg := range strings.Fields(string(out)) {
		for prefix, remedy := range banned {
			hit := pkg == prefix
			if !hit && strings.HasSuffix(prefix, "/") {
				hit = strings.HasPrefix(pkg, prefix)
			}
			if hit {
				linked = append(linked, pkg+" ("+remedy+")")
			}
		}
	}

	if len(linked) > 0 {
		t.Errorf("the root package links %d package(s) it is supposed to stay clear of, want none:\n\t%s",
			len(linked), strings.Join(linked, "\n\t"))
	}
}
