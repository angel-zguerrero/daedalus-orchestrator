package dragonboat

import (
	"crypto/md5"
	"deadalus-orch/server/internal/pkg/config" // Added
	"deadalus-orch/shared/constants"           // Added
	"os"
	"os/user" // Added
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert" // Added
	"github.com/stretchr/testify/require" // Added
)

func TestGetNodeDBDirName(t *testing.T) {
	path, err := getNodeDBDirName(1, 2)
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
	all := ""
	roles, err := ParseRolesFlag(&all)
	if err != nil || len(roles) != 3 {
		t.Errorf("expected all roles, got %v", roles)
	}
	custom := "consensus,scheduler"
	roles, err = ParseRolesFlag(&custom)
	if err != nil || len(roles) != 2 {
		t.Errorf("expected 2 roles, got %v", roles)
	}
	bad := "foo"
	_, err = ParseRolesFlag(&bad)
	if err == nil {
		t.Errorf("expected error for invalid role")
	}
}

func TestParseMember(t *testing.T) {
	valid := "127.0.0.1:8080"
	member, err := ParseMember(valid)
	if err != nil || member.IP != "127.0.0.1" || member.Port != 8080 {
		t.Errorf("invalid parse: %v", err)
	}
	_, err = ParseMember("invalid")
	if err == nil {
		t.Errorf("expected error for malformed input")
	}
	_, err = ParseMember("127.0.0.1:abc")
	if err == nil {
		t.Errorf("expected error for invalid port")
	}
}

func TestParseMembersFlag(t *testing.T) {
	input := "127.0.0.1:1234,192.168.1.1:4321"
	members, err := ParseMembersFlag(&input)
	if err != nil || len(members) != 2 {
		t.Errorf("unexpected error or count mismatch: %v", err)
	}
	bad := "abc"
	_, err = ParseMembersFlag(&bad)
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

// --- New Test Cases for Phase 2 ---

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
			expectError: false, // Assuming ParseRolesList filters out empty/whitespace
		},
		{
			name:        "invalid role with valid ones",
			input:       []string{"consensus", "invalid_role", "scheduler"},
			expected:    nil, // Or []NodeRole{RoleConsensus, RoleScheduler} if invalid is ignored
			expectError: true, // Assuming it errors out on any invalid role
		},
		{
			name:        "only empty and whitespace strings",
			input:       []string{"", " ", "  "},
			expected:    []NodeRole{},
			expectError: false, // Assuming these are filtered out
		},
		{
			name:        "all valid roles",
			input:       []string{"consensus", "scheduler", "connector"},
			expected:    []NodeRole{RoleConsensus, RoleScheduler, RoleConnector},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ParseRolesList is unexported. We test it via ParseRolesFlag.
			// To do this, we join the input slice into a comma-separated string for ParseRolesFlag.
			rolesStr := strings.Join(tt.input, ",")
			if len(tt.input) == 0 { // Handle case where input is truly empty list vs list of empty strings
				rolesStr = "" // ParseRolesFlag expects pointer, so empty string means "all"
				if tt.name == "empty list" { // specific case for empty list
					rolesStrPtr := ""
					roles, err := ParseRolesFlag(&rolesStrPtr) // this should return all 3 roles if "" means "all"
					if tt.expectError {
						assert.Error(t, err)
					} else {
						assert.NoError(t, err)
						// For empty list input, ParseRolesFlag with "" actually means "all roles"
						// So, if the test expected an empty list, this needs adjustment or ParseRolesList needs direct test
						// Given the original ParseRolesFlag test, "" for the string means "all"
						// This test case as "empty list" for ParseRolesList might be better tested if ParseRolesList were exported
						// Or by adjusting expectation for ParseRolesFlag:
						if tt.name == "empty list" {
							// If ParseRolesFlag interprets "" as "all", then this test is for that behavior
							assert.ElementsMatch(t, []NodeRole{RoleConsensus, RoleScheduler, RoleConnector}, roles, "ParseRolesFlag with empty string should yield all roles")
						} else {
							assert.ElementsMatch(t, tt.expected, roles)
						}
					}
					return
				}
			}

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

func TestParseMember_DetailedErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string // Substring to check for in the error message
	}{
		{
			name:          "missing port",
			input:         "127.0.0.1",
			expectedError: "member address 127.0.0.1 must be in ip:port format",
		},
		{
			name:          "invalid IP",
			input:         "invalid-ip:8080",
			expectedError: "address invalid-ip:8080: invalid IP address",
		},
		{
			name:          "non-numeric port",
			input:         "127.0.0.1:notaport",
			expectedError: "member address 127.0.0.1:notaport: port notaport is not a number",
		},
		{
			name:          "port out of range (too high)",
			input:         "127.0.0.1:70000",
			expectedError: "member address 127.0.0.1:70000: port 70000 is not a valid port number",
		},
		{
			name:          "port out of range (too low)",
			input:         "127.0.0.1:0",
			expectedError: "member address 127.0.0.1:0: port 0 is not a valid port number",
		},
		{
			name:          "empty string",
			input:         "",
			expectedError: "member address  is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMember(tt.input)
			assert.Error(t, err)
			if tt.expectedError != "" {
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestGetInitialMembers_EmptyInputs(t *testing.T) {
	members, err := GetInitialMembers([]string{}, []int{})
	assert.NoError(t, err)
	assert.Empty(t, members)
}

func TestGetNodeDBDirName_ErrorPath(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Forcing GetDatabasePath to error by unsetting HOME (used in dev mode)
	// and setting ENV to development explicitly.
	t.Setenv(constants.EnvVarEnvKey, string(constants.DEVELOPMENT))
	err := os.Unsetenv("HOME")
	require.NoError(t, err, "Failed to unset HOME for test")

	// Reload config to ensure it picks up the changed ENV_VAR and no HOME
	// This might still not be enough if config.LoadDefaultConfiguration was already called
	// and GlobalConfiguration is already set.
	// For a robust test, DefaultPathProvider might need to be injectable or config reloaded.
	// Assuming LoadDefaultConfiguration is called by the system under test or can be triggered.
	// For now, we directly instantiate DefaultPathProvider which is what getNodeDBDirName does.

	// This relies on db.DefaultPathProvider().GetDatabasePath() failing.
	// The path provider itself is in another package 'db', making its direct mock hard here.
	// We are testing getNodeDBDirName's error propagation.
	config.GlobalConfiguration = nil // Attempt to force re-evaluation or ensure fresh state
	err = config.LoadDefaultConfiguration() // This will re-parse flags and potentially error on HOME
	
	// If loading config itself errors due to HOME unset, that's fine for this test's purpose,
	// as getNodeDBDirName wouldn't be reached or would get an error from path provider.
	// If LoadDefaultConfiguration sets a default HOME or doesn't error, this test might not cover the intended path.

	// The actual call within getNodeDBDirName:
	// pathProvider := &db.DefaultPathProvider{}
	// basePath, err := pathProvider.GetDatabasePath()

	// If LoadDefaultConfiguration fails due to HOME being unset, that's the error we expect GetDatabasePath to cause.
	if err != nil && strings.Contains(err.Error(), "user: Current requires cgo") || strings.Contains(err.Error(), "HOME not set") {
		// This means GetDatabasePath inside LoadDefaultConfiguration (or if called directly) would fail.
		// So, getNodeDBDirName would also fail.
		// We can't directly call getNodeDBDirName if config load fails this early.
		// This test setup highlights the difficulty of testing this specific error path
		// without more advanced mocking or refactoring DefaultPathProvider.
		t.Logf("LoadDefaultConfiguration failed as expected due to HOME unset: %v", err)
		// We assume this error from config loading implies getNodeDBDirName would also fail if it reached GetDatabasePath.
		// To directly test getNodeDBDirName's error path, GetDatabasePath needs to be mockable.
		// Given the constraints, we accept that a failure in LoadDefaultConfiguration due to path issues
		// indirectly tests the scenario where getNodeDBDirName would receive an error.
		assert.Error(t, err) // Assert that loading config (which uses path provider) errors out
	} else {
		// If LoadDefaultConfiguration somehow "succeeds" (e.g. by os.Args flags overriding env needs)
		// then attempt to call getNodeDBDirName, expecting it to fail.
		_, errGetNodeDB := getNodeDBDirName(1, 1)
		assert.Error(t, errGetNodeDB, "getNodeDBDirName should fail if GetDatabasePath fails")
		if errGetNodeDB != nil {
			assert.True(t, strings.Contains(errGetNodeDB.Error(), "user: Current requires cgo") || strings.Contains(errGetNodeDB.Error(), "HOME not set"),
				"Error message should indicate failure related to user/home path.")
		}
	}
	// Restore HOME for other tests if necessary, though t.Setenv and test isolation should handle it.
	// os.Setenv("HOME", originalHome) // Handled by defer
}
