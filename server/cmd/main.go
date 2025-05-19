package main

import (
	"deadalus-orch/server/internal/app"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"flag"
	"log"
	"strconv"
)

func main() {
	roleFlag := flag.String("role", "", "Comma-separated node roles: consensus, scheduler, connector")
	membersFlag := flag.String("members", "", "Cluster members as ip:port,ip:port,...")
	addr := flag.String("addr", "", "Nodehost address (ip:port)")
	replicaID := flag.Int("replica", 1, "Nodehost replica")
	flag.Parse()

	roles, err := dragonboat.ParseRolesFlag(roleFlag)

	if err != nil {
		log.Fatal("❌ Failed Parsing roles", err)
	}

	members, err := dragonboat.ParseMembersFlag(membersFlag)
	if err != nil {
		log.Fatal("Error parsing members:", err)
		return
	}

	if *addr == "" {
		localIp := dragonboat.LocalDefaultHost
		*addr = localIp + ":" + strconv.Itoa(dragonboat.LocalDefaultPort)
	}

	selfMember, err := dragonboat.ParseMember(*addr)
	if err != nil {
		log.Fatal("Self mermber parsing error:", err)
		return
	}
	app.Run(*replicaID, roles, selfMember, members)
}
