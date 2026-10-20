package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// skipUnlessPermissionsBind skips a test that relies on a permission
// denial where file modes do not bind: on Windows, and for root.
func skipUnlessPermissionsBind(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes do not deny access here")
	}
}

// unreadableFixture writes a file the test process may not read and
// returns its path.
func unreadableFixture(t *testing.T, dir, name string) string {
	t.Helper()
	skipUnlessPermissionsBind(t)
	path := writeViewFixture(t, dir, name, "secret\n")
	require.NoError(t, os.Chmod(path, 0o000))
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	return path
}

// unwritableDir creates a directory the test process may list but not
// create entries in, and returns its path.
func unwritableDir(t *testing.T, parent, name string) string {
	t.Helper()
	skipUnlessPermissionsBind(t)
	dir := filepath.Join(parent, name)
	require.NoError(t, os.Mkdir(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	return dir
}
