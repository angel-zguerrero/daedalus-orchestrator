package dragonboat

import (
	"bytes"
	"crypto/md5"
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/binary"
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

	"github.com/rs/zerolog/log"
)

// getNodeDBDirName constructs the database directory path for a specific Raft node.
// The path is typically <base_database_path>/<clusterID>_<nodeID>.
//
// Parameters:
//   - clusterID: The ID of the Raft cluster (shard).
//   - nodeID: The ID of the Raft node (replica).
//
// Returns:
//   - The constructed directory path as a string.
//   - An error if the base database path cannot be determined.
func getNodeDBDirName(clusterID uint64, nodeID uint64) (string, error) {
	part := fmt.Sprintf("%d_%d", clusterID, nodeID)
	database_path, err := (&db.DefaultPathProvider{}).GetDatabasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(database_path, part), nil
}

// syncDir flushes directory changes to the underlying storage.
// On Windows, this is a no-op. On other systems, it opens the directory
// and calls Sync on the file descriptor.
//
// Parameters:
//   - dir: The path to the directory to be synced.
//
// Returns:
//   - An error if stating the directory, opening it, or syncing it fails.
// Panics if `dir` is not a directory.
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

// createNodeDataDir creates a directory and syncs its parent directory.
// This is often used for creating directories that will hold database files
// to ensure the directory entry is persisted.
//
// Parameters:
//   - dir: The path of the directory to create.
//
// Returns:
//   - An error if creating the directory or syncing its parent fails.
func createNodeDataDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dir))
}

// isNewRun checks if the current execution is a new run for a node by
// looking for the existence of the CurrentDBFilename in the specified directory.
//
// Parameters:
//   - dir: The node's base data directory.
//
// Returns:
//   - True if CurrentDBFilename does not exist, indicating a new run.
//   - False otherwise.
func isNewRun(dir string) bool {
	fp := filepath.Join(dir, CurrentDBFilename)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return true
	}
	return false
}

// getCurrentDBDirName reads the CurrentDBFilename file to get the name of the
// current active database directory. The file content is expected to be
// an 8-byte MD5 checksum followed by the directory name.
//
// Parameters:
//   - dir: The node's base data directory where CurrentDBFilename is located.
//
// Returns:
//   - The name of the current database directory.
//   - An error if opening or reading the file fails, or if the content is corrupted.
// Panics if the file content is too short or if the checksum doesn't match.
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

// cleanupNodeDataDir removes old or temporary database directories.
// It first removes any directory named UpdatingDBFilename.
// Then, it reads the current DB directory name and deletes any other
// subdirectories in the node's base data directory.
//
// Parameters:
//   - dir: The node's base data directory.
//
// Returns:
//   - An error if reading the current DB directory name fails, listing files fails,
//     or removing old directories fails.
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
		log.Info().Msgf("dbdir %s, fi.name %s, dir %s", dbdir, fi.Name(), dir)
		toDelete := filepath.Join(dir, fi.Name())
		if toDelete != dbdir { // delete old databases
			log.Info().Msgf("removing %s", toDelete)
			if err := os.RemoveAll(toDelete); err != nil {
				return err
			}
		}
	}
	return nil
}

// getNewRandomDBDirName generates a new random directory name for a database instance.
// The name is typically based on a random number and the current nanosecond timestamp
// to ensure uniqueness.
//
// Parameters:
//   - dir: The parent directory where the new random directory will conceptually reside (used for joining).
//
// Returns:
//   - A string representing the full path to the new random directory.
func getNewRandomDBDirName(dir string) string {
	part := "%d_%d"
	rn := rand.Uint64()
	ct := time.Now().UnixNano()
	return filepath.Join(dir, fmt.Sprintf(part, rn, ct))
}

// saveCurrentDBDirName saves the name of the active database directory to the
// UpdatingDBFilename file. It prepends an 8-byte MD5 checksum of the directory name.
// This file acts as a temporary placeholder before being renamed to CurrentDBFilename.
//
// Parameters:
//   - dir: The node's base data directory.
//   - dbdir: The name of the current active database directory to save.
//
// Returns:
//   - An error if writing the checksum, directory name, or syncing the file/directory fails.
// Panics if closing the file or syncing the parent directory fails.
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

// replaceCurrentDBFile renames UpdatingDBFilename to CurrentDBFilename,
// effectively making the database directory listed in UpdatingDBFilename the active one.
// It then syncs the parent directory.
//
// Parameters:
//   - dir: The node's base data directory.
//
// Returns:
//   - An error if renaming the file or syncing the directory fails.
func replaceCurrentDBFile(dir string) error {
	fp := filepath.Join(dir, CurrentDBFilename)
	tmpFp := filepath.Join(dir, UpdatingDBFilename)
	if err := os.Rename(tmpFp, fp); err != nil {
		return err
	}
	return syncDir(dir)
}

// ParseRolesList converts a list of role strings into a slice of NodeRole types.
// It validates each role against a predefined set of valid roles.
//
// Parameters:
//   - list: A slice of strings, where each string is a potential node role.
//
// Returns:
//   - A slice of NodeRole.
//   - An error if any role string is invalid.
func ParseRolesList(list []string) ([]NodeRole, error) {
	var validRoles = map[string]bool{
		string(RoleConsensus): true,
		string(RoleScheduler): true,
		string(RoleConnector): true,
		string(RoleAdmin):     true,
	}

	roles := make([]NodeRole, 0, len(list))
	for _, r := range list {
		role := strings.TrimSpace(r)
		if !validRoles[role] {
			return nil, fmt.Errorf("invalid role: %s. Valid roles are: consensus, scheduler, connector, admin", role)
		}
		roles = append(roles, NodeRole(role))
	}
	return roles, nil
}

// ParseRolesFlag parses a comma-separated string of roles into a slice of NodeRole.
// If the input string is empty, it returns a default list of all roles.
//
// Parameters:
//   - roleSeparateComma: A pointer to a string containing comma-separated roles.
//
// Returns:
//   - A slice of NodeRole.
//   - An error if any role is invalid (via ParseRolesList).
func ParseRolesFlag(roleSeparateComma *string) ([]NodeRole, error) {
	if *roleSeparateComma == "" {
		return ParseRolesList([]string{
			string(RoleConsensus),
			string(RoleScheduler),
			string(RoleConnector),
			string(RoleAdmin),
		})
	}

	partsRaw := strings.Split(*roleSeparateComma, ",")
	parts := make([]string, 0, len(partsRaw))
	for _, part := range partsRaw {
		partTrimmed := strings.TrimSpace(part)
		if partTrimmed != "" {
			parts = append(parts, partTrimmed)
		}
	}

	return ParseRolesList(parts)
}

// ParseMember parses a string representation of a node member (e.g., "127.0.0.1:7000")
// into a Member struct. It validates the IP address and port.
//
// Parameters:
//   - raw: The string representation of the member.
//
// Returns:
//   - A Member struct.
//   - An error if the format is invalid, IP is invalid, or port is out of range.
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

// ParseMembersFlag parses a comma-separated string of members into a slice of Member structs.
// If the input string is empty, it returns an empty slice.
//
// Parameters:
//   - membersFlag: A pointer to a string containing comma-separated member addresses.
//
// Returns:
//   - A slice of Member structs.
//   - An error if any member string cannot be parsed (via ParseMember).
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

// ToInitialMembersMap converts a slice of Member structs into a map suitable for
// Dragonboat's initial members configuration. The map keys are node IDs (starting from 1)
// and values are their Raft addresses.
//
// Parameters:
//   - members: A slice of Member structs.
//
// Returns:
//   - A map where keys are uint64 node IDs and values are string Raft addresses.
func ToInitialMembersMap(members []Member) map[uint64]string {
	initialMembers := make(map[uint64]string, len(members))
	for i, m := range members {
		nodeID := uint64(i + 1) // Node IDs typically start from 1.
		addr := MemmberToAddr(m)
		initialMembers[nodeID] = addr
	}
	return initialMembers
}

// MemmberToAddr converts a Member struct into its string Raft address representation ("IP:Port").
//
// Parameters:
//   - member: The Member struct to convert.
//
// Returns:
//   - The string representation of the member's address.
func MemmberToAddr(member Member) string {
	return fmt.Sprintf("%s:%d", member.IP, member.Port)
}

// MergeUniqueMembers combines a 'self' Member with a slice of 'other' Members,
// ensuring the 'self' member is not already present in 'others'.
//
// Parameters:
//   - self: The primary Member.
//   - others: A slice of other Member structs.
//
// Returns:
//   - A new slice containing 'self' followed by 'others'.
//   - An error if 'self' is found to be a duplicate within 'others'.
func MergeUniqueMembers(self Member, others []Member) ([]Member, error) {
	for _, m := range others {
		if m.IP == self.IP && m.Port == self.Port {
			return nil, fmt.Errorf("selfMember (%s:%d) already exists in otherMembers", self.IP, self.Port)
		}
	}
	combined := append([]Member{self}, others...)
	return combined, nil
}

// IsMemberInMemberArray checks if a given Member is present in a slice of Member structs.
//
// Parameters:
//   - selfMember: The Member to search for.
//   - initialMembers: The slice of Member structs to search within.
//
// Returns:
//   - True if selfMember is found in initialMembers, false otherwise.
func IsMemberInMemberArray(selfMember Member, initialMembers []Member) bool {
	for _, member := range initialMembers {
		if member == selfMember {
			return true
		}
	}
	return false
}

// ContainsRole checks if a specific NodeRole is present in a slice of NodeRole.
//
// Parameters:
//   - roles: A slice of NodeRole.
//   - role: The NodeRole to search for.
//
// Returns:
//   - True if the role is found in the slice, false otherwise.
func ContainsRole(roles []NodeRole, role NodeRole) bool {
	for _, s := range roles {
		if s == role {
			return true
		}
	}
	return false
}

// Int64ToBytes converts an int64 value to its 8-byte big-endian representation.
//
// Parameters:
//   - n: The int64 value to convert.
//
// Returns:
//   - A byte slice of length 8 representing the int64.
func Int64ToBytes(n int64) []byte {
	buf := make([]byte, 8) // int64 = 8 bytes
	binary.BigEndian.PutUint64(buf, uint64(n))
	return buf
}

// GetInitialMembers creates a slice of Member structs from parallel slices of IPs and ports.
// It's a utility for constructing member lists when IP and port information is separate.
//
// Parameters:
//   - InitialMemberIps: A slice of IP address strings.
//   - InitialMemberPorts: A slice of port integers.
//
// Returns:
//   - A slice of Member structs.
//   - An error if the lengths of the IP and port slices do not match.
func GetInitialMembers(InitialMemberIps []string, InitialMemberPorts []int) ([]Member, error) {
	if len(InitialMemberIps) != len(InitialMemberPorts) {
		return nil, fmt.Errorf("mismatched lengths: %d IPs vs %d ports", len(InitialMemberIps), len(InitialMemberPorts))
	}

	members := make([]Member, 0, len(InitialMemberIps))
	for i := 0; i < len(InitialMemberIps); i++ {
		members = append(members, Member{
			IP:   InitialMemberIps[i],
			Port: InitialMemberPorts[i],
		})
	}

	return members, nil
}
