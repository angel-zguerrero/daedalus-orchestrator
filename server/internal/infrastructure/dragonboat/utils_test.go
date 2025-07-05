package dragonboat

import (
	"crypto/md5"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestPathProvider struct {
	Path string
}

func (d TestPathProvider) GetDatabasePath() (string, error) {
	return d.Path, nil
}

func TestGetNodeDBDirName(t *testing.T) {
	path, err := getNodeDBDirName(1, 2, &TestPathProvider{Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "1_2") {
		t.Errorf("expected path to contain clusterID and nodeID: %s", path)
	}
}

func TestSyncDir(t *testing.T) {
	t.Run("syncs directory", func(t *testing.T) {
		dir := t.TempDir()
		err := syncDir(dir)
		if runtime.GOOS != "windows" && err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails on non-existing dir", func(t *testing.T) {
		err := syncDir("/non/existing/dir")
		if err == nil {
			t.Errorf("expected error for non-existing dir")
		}
	})

	t.Run("panics on non-dir", func(t *testing.T) {
		tmpfile := filepath.Join(t.TempDir(), "file.txt")
		_ = os.WriteFile(tmpfile, []byte("data"), 0644)

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic on file sync")
			}
		}()
		_ = syncDir(tmpfile)
	})
}

func TestCreateNodeDataDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir")
	err := createNodeDataDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("directory was not created")
	}
}

func TestIsNewRun(t *testing.T) {
	dir := t.TempDir()
	if !isNewRun(dir) {
		t.Errorf("expected isNewRun to return true on empty dir")
	}
	fp := filepath.Join(dir, CurrentDBFilename)
	_ = os.WriteFile(fp, []byte("dummy"), 0644)
	if isNewRun(dir) {
		t.Errorf("expected isNewRun to return false after creating CURRENT")
	}
}

func TestGetCurrentDBDirName(t *testing.T) {
	dir := t.TempDir()
	dbdir := "abc123"
	h := md5.New()
	h.Write([]byte(dbdir))
	crc := h.Sum(nil)[:8]
	content := append(crc, []byte(dbdir)...)
	_ = os.WriteFile(filepath.Join(dir, CurrentDBFilename), content, 0644)

	res, err := getCurrentDBDirName(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res != dbdir {
		t.Errorf("expected dbdir %s, got %s", dbdir, res)
	}
}

func TestCleanupNodeDataDir(t *testing.T) {
	dir := t.TempDir()
	dbdir := "keep"
	dbPath := filepath.Join(dir, dbdir)
	_ = os.Mkdir(dbPath, 0755)

	h := md5.New()
	h.Write([]byte(dbPath))
	crc := h.Sum(nil)[:8]
	content := append(crc, []byte(dbPath)...)
	_ = os.WriteFile(filepath.Join(dir, CurrentDBFilename), content, 0644)

	_ = os.Mkdir(filepath.Join(dir, "old1"), 0755)
	_ = os.Mkdir(filepath.Join(dir, "old2"), 0755)
	err := cleanupNodeDataDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old1")); !os.IsNotExist(err) {
		t.Errorf("old1 dir should be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, dbdir)); err != nil {
		t.Errorf("dbdir %s should not be deleted", dbdir)
	}
}

func TestGetNewRandomDBDirName(t *testing.T) {
	name := getNewRandomDBDirName("path")
	if !strings.Contains(name, "path") {
		t.Errorf("expected path to be included")
	}
}

func TestSaveAndReplaceCurrentDB(t *testing.T) {
	dir := t.TempDir()
	dbdir := "saved_dir"
	err := saveCurrentDBDirName(dir, dbdir)
	if err != nil {
		t.Fatal(err)
	}
	err = replaceCurrentDBFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	res, err := getCurrentDBDirName(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res != dbdir {
		t.Errorf("expected %s, got %s", dbdir, res)
	}
}

func TestParseRolesFlag(t *testing.T) {
	// Test case 1: Empty role string, should return all default roles including Admin
	all := ""
	roles, err := ParseRolesFlag(&all)
	assert.NoError(t, err)
	assert.Len(t, roles, 4, "Expected all default roles including admin")
	assert.Contains(t, roles, RoleConsensus)
	assert.Contains(t, roles, RoleScheduler)
	assert.Contains(t, roles, RoleConnector)
	assert.Contains(t, roles, RoleAdmin)

	// Test case 2: Custom roles
	custom := "consensus,scheduler"
	roles, err = ParseRolesFlag(&custom)
	assert.NoError(t, err)
	assert.Len(t, roles, 2, "Expected 2 custom roles")
	assert.Contains(t, roles, RoleConsensus)
	assert.Contains(t, roles, RoleScheduler)

	// Test case 3: Only admin role
	adminOnly := "admin"
	roles, err = ParseRolesFlag(&adminOnly)
	assert.NoError(t, err)
	assert.Len(t, roles, 1, "Expected 1 role (admin)")
	assert.Contains(t, roles, RoleAdmin)

	// Test case 4: Custom roles including admin
	customWithAdmin := "connector,admin"
	roles, err = ParseRolesFlag(&customWithAdmin)
	assert.NoError(t, err)
	assert.Len(t, roles, 2, "Expected 2 custom roles including admin")
	assert.Contains(t, roles, RoleConnector)
	assert.Contains(t, roles, RoleAdmin)

	// Test case 5: Invalid role
	bad := "foo"
	_, err = ParseRolesFlag(&bad)
	assert.Error(t, err, "Expected error for invalid role")
	assert.Contains(t, err.Error(), "invalid role: foo. Valid roles are: consensus, scheduler, connector, admin", "Error message should list valid roles including admin")
}

func TestParseMember(t *testing.T) {

	validMember := "127.0.0.1:r1"
	member, err := ParseMember(validMember, 7999)
	if err != nil || member.IP != "127.0.0.1" || member.Port != 8000 {
		t.Errorf("invalid parse: %v", err)
	}
	_, err = ParseMember("invalid", 9)
	if err == nil {
		t.Errorf("expected error for malformed input")
	}
	_, err = ParseMember("127.0.0.1:rt", 9)
	if err == nil {
		t.Errorf("expected error for invalid port")
	}
}

func TestParseMembersFlag(t *testing.T) {
	input := "127.0.0.1:r1,192.168.1.1:r2"
	members, err := ParseMembersFlag(&input, 7990)
	if err != nil || len(members) != 2 {
		t.Errorf("unexpected error or count mismatch: %v", err)
	}
	bad := "abc"
	_, err = ParseMembersFlag(&bad, 7990)
	if err == nil {
		t.Errorf("expected error for invalid member")
	}
}

func TestToInitialMembers(t *testing.T) {
	members := []Member{
		{IP: "127.0.0.1", Port: 1},
		{IP: "127.0.0.2", Port: 2},
	}
	res := ToInitialMembersMap(members)
	if len(res) != 2 {
		t.Errorf("expected 2 members, got %d", len(res))
	}
}

func TestMemmberToAddr(t *testing.T) {
	member := Member{IP: "1.1.1.1", Port: 80}
	addr := MemmberToAddr(member)
	if addr != "1.1.1.1:80" {
		t.Errorf("unexpected address: %s", addr)
	}
}

func TestMergeUniqueMembers(t *testing.T) {
	self := Member{IP: "127.0.0.1", Port: 1234}
	others := []Member{{IP: "10.0.0.1", Port: 9999}}
	res, err := MergeUniqueMembers(self, others)
	if err != nil || len(res) != 2 {
		t.Errorf("expected merge without error")
	}

	conflict := []Member{{IP: "127.0.0.1", Port: 1234}}
	_, err = MergeUniqueMembers(self, conflict)
	if err == nil {
		t.Errorf("expected conflict error")
	}
}

func TestParseRolesList_ExtendedCases(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    []NodeRole
		expectError bool
	}{
		{
			name:        "empty list",
			input:       []string{},
			expected:    []NodeRole{},
			expectError: false,
		},
		{
			name:        "valid roles with empty and whitespace strings",
			input:       []string{"consensus", "", " ", "scheduler", "  "},
			expected:    []NodeRole{RoleConsensus, RoleScheduler},
			expectError: false,
		},
		{
			name:        "invalid role with valid ones",
			input:       []string{"consensus", "invalid_role", "scheduler"},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "only empty and whitespace strings",
			input:       []string{"", " ", "  "},
			expected:    []NodeRole{},
			expectError: false,
		},
		{
			name:        "all valid roles",
			input:       []string{"consensus", "scheduler", "connector", "admin"},
			expected:    []NodeRole{RoleConsensus, RoleScheduler, RoleConnector, RoleAdmin},
			expectError: false,
		},
		{
			name:        "admin role alone",
			input:       []string{"admin"},
			expected:    []NodeRole{RoleAdmin},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rolesStr := strings.Join(tt.input, ",")
			// Special handling for the "empty list" test case, as ParseRolesFlag has specific behavior for empty input
			if tt.name == "empty list" {
				emptyStr := ""
				roles, err := ParseRolesFlag(&emptyStr) // ParseRolesFlag is tested, not ParseRolesList
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					// For an empty input string, ParseRolesFlag should return all default roles
					assert.ElementsMatch(t, []NodeRole{RoleConsensus, RoleScheduler, RoleConnector, RoleAdmin}, roles, "ParseRolesFlag with empty string should yield all default roles")
				}
				return // Skip further processing for this specific test case
			}

			// For other test cases, we are effectively testing ParseRolesList via ParseRolesFlag
			// by providing a non-empty, comma-separated string.
			// ParseRolesList itself is not directly invoked here if tt.input is empty due to the above check,
			// which is fine as ParseRolesFlag's default behavior is what we want to test for empty input.

			roles, err := ParseRolesFlag(&rolesStr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expected, roles)
			}
		})
	}
}

func TestGetInitialMembers_EmptyInputs(t *testing.T) {
	members, err := GetInitialMembers([]string{}, []int{})
	assert.NoError(t, err)
	assert.Empty(t, members)
}
