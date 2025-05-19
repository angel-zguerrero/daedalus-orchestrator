package dragonboat

import (
	"crypto/md5"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	res := ToInitialMembers(members)
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

func TestGetLocalIP(t *testing.T) {
	ip, err := GetLocalIP()
	if err != nil || ip != "127.0.0.1" {
		t.Errorf("unexpected IP: %v", err)
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
