package dragonboat

import (
	"bytes"
	"crypto/md5"
	"deadalus-orch/server/internal/infrastructure/db"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func getNodeDBDirName(clusterID uint64, nodeID uint64) (string, error) {
	part := fmt.Sprintf("%d_%d", clusterID, nodeID)
	database_path, err := (&db.DefaultPathProvider{}).GetDatabasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(database_path, part), nil
}

func syncDir(dir string) (err error) { // good practice
	if runtime.GOOS == "windows" {
		return nil
	}
	fileInfo, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		panic("not a dir")
	}
	df, err := os.Open(filepath.Clean(dir))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := df.Close(); err == nil {
			err = cerr
		}
	}()
	return df.Sync()
}

func createNodeDataDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dir))
}

func isNewRun(dir string) bool {
	fp := filepath.Join(dir, CurrentDBFilename)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return true
	}
	return false
}
func getCurrentDBDirName(dir string) (string, error) {
	fp := filepath.Join(dir, CurrentDBFilename)
	f, err := os.OpenFile(fp, os.O_RDONLY, 0755)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	data, err := ioutil.ReadAll(f) // CURRENT FILES WILL BE SMALL
	if err != nil {
		return "", err
	}
	if len(data) <= 8 {
		panic("corrupted content")
	}
	crc := data[:8]
	content := data[8:]
	h := md5.New()
	if _, err := h.Write(content); err != nil {
		return "", err
	}
	if !bytes.Equal(crc, h.Sum(nil)[:8]) {
		panic("corrupted content with not matched crc")
	}
	return string(content), nil
}
func cleanupNodeDataDir(dir string) error {
	os.RemoveAll(filepath.Join(dir, UpdatingDBFilename))
	dbdir, err := getCurrentDBDirName(dir)
	if err != nil {
		return err
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range files {
		if !fi.IsDir() {
			continue
		}
		fmt.Printf("dbdir %s, fi.name %s, dir %s\n", dbdir, fi.Name(), dir)
		toDelete := filepath.Join(dir, fi.Name())
		if toDelete != dbdir { // delete old databases
			fmt.Printf("removing %s\n", toDelete)
			if err := os.RemoveAll(toDelete); err != nil {
				return err
			}
		}
	}
	return nil
}

func getNewRandomDBDirName(dir string) string {
	part := "%d_%d"
	rn := rand.Uint64()
	ct := time.Now().UnixNano()
	return filepath.Join(dir, fmt.Sprintf(part, rn, ct))
}
func saveCurrentDBDirName(dir string, dbdir string) error {
	h := md5.New()
	if _, err := h.Write([]byte(dbdir)); err != nil {
		return err
	}
	fp := filepath.Join(dir, UpdatingDBFilename)
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
		if err := syncDir(dir); err != nil {
			panic(err)
		}
	}()
	if _, err := f.Write(h.Sum(nil)[:8]); err != nil {
		return err
	}
	if _, err := f.Write([]byte(dbdir)); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}
func replaceCurrentDBFile(dir string) error {
	fp := filepath.Join(dir, CurrentDBFilename)
	tmpFp := filepath.Join(dir, UpdatingDBFilename)
	if err := os.Rename(tmpFp, fp); err != nil {
		return err
	}
	return syncDir(dir)
}

func ParseRolesFlag(roleSeparateComma *string) ([]string, error) {
	var validRoles = map[string]bool{
		string(RoleConsensus): true,
		string(RoleScheduler): true,
		string(RoleConnector): true,
	}

	if *roleSeparateComma == "" {
		return []string{
			string(RoleConsensus),
			string(RoleScheduler),
			string(RoleConnector),
		}, nil
	}

	parts := strings.Split(*roleSeparateComma, ",")
	roles := make([]string, 0, len(parts))
	for _, r := range parts {
		role := strings.TrimSpace(r)
		if !validRoles[role] {
			return nil, fmt.Errorf("invalid role: %s. Valid roles are: consensus, scheduler, connector", role)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func ParseMember(raw string) (Member, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Member{}, errors.New("empty member entry")
	}

	host, portStr, err := net.SplitHostPort(raw)
	if err != nil {
		return Member{}, fmt.Errorf("invalid member format '%s': %v", raw, err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return Member{}, fmt.Errorf("invalid IP address: %s", host)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return Member{}, fmt.Errorf("invalid port: %s", portStr)
	}

	return Member{
		IP:   ip.String(),
		Port: port,
	}, nil
}

func ParseMembersFlag(membersFlag *string) ([]Member, error) {
	if *membersFlag == "" {
		return []Member{}, nil
	}

	rawParts := strings.Split(*membersFlag, ",")
	members := make([]Member, 0, len(rawParts))

	for _, raw := range rawParts {
		member, err := ParseMember(raw)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

func ToInitialMembersMap(members []Member) map[uint64]string {
	initialMembers := make(map[uint64]string, len(members))
	for i, m := range members {
		nodeID := uint64(i + 1)
		addr := MemmberToAddr(m)
		initialMembers[nodeID] = addr
	}
	return initialMembers
}

func MemmberToAddr(member Member) string {
	return fmt.Sprintf("%s:%d", member.IP, member.Port)
}

func MergeUniqueMembers(self Member, others []Member) ([]Member, error) {
	for _, m := range others {
		if m.IP == self.IP && m.Port == self.Port {
			return nil, fmt.Errorf("selfMember (%s:%d) already exists in otherMembers", self.IP, self.Port)
		}
	}
	combined := append([]Member{self}, others...)
	return combined, nil
}

func IsMemberInMemberArray(selfMember Member, initialMembers []Member) bool {
	for _, member := range initialMembers {
		if member == selfMember {
			return true
		}
	}
	return false
}
