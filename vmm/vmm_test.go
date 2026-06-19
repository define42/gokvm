package vmm

import (
	"strings"
	"testing"
)

func TestISOCmdlineAddsCompatibilityParams(t *testing.T) {
	t.Parallel()

	cmdline := isoCmdline("loglevel=3 cde")

	for _, want := range []string{
		"console=tty0",
		"console=ttyS0",
		"earlyprintk=serial",
		"noapic",
		"noacpi",
		"notsc",
		"pci=realloc=off",
		"virtio_pci.force_legacy=1",
		"loglevel=3",
		"cde",
	} {
		if !hasField(cmdline, want) {
			t.Fatalf("cmdline %q is missing %q", cmdline, want)
		}
	}
}

func TestISOCmdlineDoesNotDuplicateExistingParams(t *testing.T) {
	t.Parallel()

	cmdline := isoCmdline("console=ttyS0 pci=nomsi custom=1")

	if got := countField(cmdline, "console=ttyS0"); got != 1 {
		t.Fatalf("console=ttyS0 count: got %d, want 1 in %q", got, cmdline)
	}

	if hasField(cmdline, "pci=realloc=off") {
		t.Fatalf("cmdline %q should not add a second pci= parameter", cmdline)
	}

	if !hasField(cmdline, "pci=nomsi") {
		t.Fatalf("cmdline %q should keep ISO pci= parameter", cmdline)
	}
}

func hasField(s, want string) bool {
	for _, field := range strings.Fields(s) {
		if field == want {
			return true
		}
	}

	return false
}

func countField(s, want string) int {
	count := 0
	for _, field := range strings.Fields(s) {
		if field == want {
			count++
		}
	}

	return count
}
