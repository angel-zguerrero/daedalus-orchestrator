package config_test

import (
	"deadalus-orch/server/internal/pkg/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigFromMapToConfig_AllFieldsCopied(t *testing.T) {
	input := config.ConfigFromMap{
		ConnectorPortVal:      8080,
		DefaultRootUserVal:    "testuser",
		DefaultRootPasswordVal: "testpass",
		ReplicaIDVal:          123,
		RolesVal:              "scheduler,connector",
		SelfMemberAddrVal:     "127.0.0.1:7001",
		InitialMembersVal:     "127.0.0.1:7001,127.0.0.1:7002",
		JoinVal:               true,
		TTLInternalErrorVal:   3600,
	}

	expected := &config.Config{
		ConnectorPort:      8080,
		DefaultRootUser:    "testuser",
		DefaultRootPassword: "testpass",
		ReplicaID:          123,
		Roles:              "scheduler,connector",
		SelfMemberAddr:     "127.0.0.1:7001",
		InitialMembers:     "127.0.0.1:7001,127.0.0.1:7002",
		Join:               true,
		TTLInternalError:   3600,
	}

	actual := config.ConfigFromMapToConfig(input)

	assert.Equal(t, expected.ConnectorPort, actual.ConnectorPort, "ConnectorPort should match")
	assert.Equal(t, expected.DefaultRootUser, actual.DefaultRootUser, "DefaultRootUser should match")
	assert.Equal(t, expected.DefaultRootPassword, actual.DefaultRootPassword, "DefaultRootPassword should match")
	assert.Equal(t, expected.ReplicaID, actual.ReplicaID, "ReplicaID should match")
	assert.Equal(t, expected.Roles, actual.Roles, "Roles should match")
	assert.Equal(t, expected.SelfMemberAddr, actual.SelfMemberAddr, "SelfMemberAddr should match")
	assert.Equal(t, expected.InitialMembers, actual.InitialMembers, "InitialMembers should match")
	assert.Equal(t, expected.Join, actual.Join, "Join should match")
	assert.Equal(t, expected.TTLInternalError, actual.TTLInternalError, "TTLInternalError should match")
}
