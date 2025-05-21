package main

import (
	"deadalus-orch/server/internal/app"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"flag"
	"log"
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

	roles, err := dragonboat.ParseRolesFlag(roleFlag)
	if err != nil {
		log.Fatal("❌ Failed Parsing roles:", err)
	}

	initialMembers, err := dragonboat.ParseMembersFlag(initialMembersFlag)
	if err != nil {
		log.Fatal("❌ Error parsing initial members:", err)
		return
	}

	if *join && len(initialMembers) > 0 {
		log.Println("⚠️ Do not use 'initial-members' when joining an existing cluster.")
		return
	}

	if !*join && len(initialMembers) == 0 && *addr != "" {
		log.Fatal("❌ 'initial-members' must be provided when creating a new cluster (with addr set).")
		return
	}

	if *addr == "" && len(initialMembers) > 0 {
		log.Fatal("❌ 'addr' must be provided when initializing a node in a multi-node cluster.")
		return
	}

	if *replicaID == 0 && len(initialMembers) > 0 {
		log.Fatal("❌ 'replica' must be specified when creating a new cluster.")
		return
	}

	// 👇 Consolidación de lógica default para addr y replica
	appendSelfToInitialMembers := false
	if *addr == "" {
		localIP := dragonboat.LocalDefaultHost
		*addr = localIP + ":" + strconv.Itoa(dragonboat.LocalDefaultPort)
		appendSelfToInitialMembers = true

		if *replicaID == 0 && len(initialMembers) == 0 {
			*replicaID = 1
		}
	}

	// 👇 Parseo justo después de determinar addr
	selfMember, err := dragonboat.ParseMember(*addr)
	if err != nil {
		log.Fatal("❌ Failed to parse self member:", err)
		return
	}

	if appendSelfToInitialMembers {
		initialMembers = append(initialMembers, selfMember)
	}

	if !dragonboat.IsMemberInMemberArray(selfMember, initialMembers) {
		log.Fatalf("❌ This node (%s) must be present in initial-members: %v", selfMember.IP, initialMembers)
	}

	app.Run(*replicaID, roles, selfMember, initialMembers)
	<-stop
}
