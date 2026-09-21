package version

import (
	"os/exec"
	"strings"
	"testing"
)

// TestRootPackageDoesNotDependOnFiber is the guard for the reason fiberadapter
// exists. Fiber support was moved out so that a service on net/http, Echo, Gin
// or chi does not link Fiber -- and with it fasthttp -- into its binary, which
// is dead weight and CVE surface it never asked for.
//
// Nothing else notices if that regresses: adding `import "github.com/gofiber/
// fiber/v3"` back to a root-package file compiles, passes every other test and
// quietly puts 25 packages back into every importer's binary. This test is
// what fails instead.
func TestRootPackageDoesNotDependOnFiber(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH; cannot inspect the import graph")
	}

	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps .: %v\n%s", err, out)
	}

	banned := []string{"github.com/gofiber/", "github.com/valyala/fasthttp"}
	var linked []string
	for _, pkg := range strings.Fields(string(out)) {
		for _, prefix := range banned {
			if strings.HasPrefix(pkg, prefix) {
				linked = append(linked, pkg)
			}
		}
	}

	if len(linked) > 0 {
		t.Errorf("the root package pulls in %d Fiber/fasthttp package(s), want none:\n\t%s",
			len(linked), strings.Join(linked, "\n\t"))
	}
}
