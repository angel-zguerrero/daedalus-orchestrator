package main

import (
	"deadalus-orch/server/internal/app"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"flag"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	roleFlag := flag.String("role", "", "Comma-separated node roles: consensus, scheduler, connector")
	initialMembersFlag := flag.String("initial-members", "", "Cluster initial members as ip:port,ip:port,...")
	addr := flag.String("addr", "", "Nodehost address (ip:port)")
	join := flag.Bool("join", false, "Joining a new node")
	replicaID := flag.Int("replica", 0, "Nodehost replica")
	flag.Parse()

	// Parse roles
	roles, err := dragonboat.ParseRolesFlag(roleFlag)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Failed Parsing roles")
	}

	// Parse addr early (even for join mode)
	if *addr == "" {
		localIP := dragonboat.LocalDefaultHost
		*addr = localIP + ":" + strconv.Itoa(dragonboat.LocalDefaultPort)
	}

	selfMember, err := dragonboat.ParseMember(*addr)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Failed to parse self member")
	}

	var initialMembers []dragonboat.Member

	if *join {
		// 🚫 No se debe usar initial-members cuando se une a un clúster
		if *initialMembersFlag != "" {
			log.Fatal().Msg("❌ Cannot use --initial-members when --join is set to true.")
		}
		if *replicaID == 0 {
			log.Fatal().Msg("❌ Must specify --replica when joining a cluster.")
		}
	} else {
		// ✅ initial-members requerido cuando NO se está uniendo
		if *initialMembersFlag == "" {
			log.Fatal().Msg("❌ Must provide --initial-members when creating a new cluster.")
		}

		initialMembers, err = dragonboat.ParseMembersFlag(initialMembersFlag)
		if err != nil {
			log.Fatal().Err(err).Msg("❌ Error parsing initial members")
		}

		// Agregar self si no está incluido explícitamente
		if !dragonboat.IsMemberInMemberArray(selfMember, initialMembers) {
			log.Fatal().Msgf("❌ This node (%s) must be present in initial-members: %v", selfMember.IP, initialMembers)
		}

		if *replicaID == 0 {
			log.Fatal().Msg("❌ Must specify --replica when creating a new cluster.")
		}
		if !dragonboat.ContainsRole(roles, dragonboat.RoleConsensus) {
			log.Fatal().Msg("❌ The role 'consensus' is required when creating a new cluster.")
		}
	}

	// Ejecutar app
	app.Run(*replicaID, roles, selfMember, initialMembers, *join)
	<-stop
}
