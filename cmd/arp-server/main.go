// Command arp-server runs the reference Agent Registry Protocol (ARP) gRPC
// server. By default it uses a process-exec backend; with -seed it uses an
// in-process mock backend and installs demo fixtures (a project, the arp-test
// workspace, and echo-agent-001 / crush-agent-001) so the gRPC conformance
// suite can run end-to-end without external agents.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/aleksclark/spec-torture/arp/backend"
	"github.com/aleksclark/spec-torture/arp/server"
	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

func main() {
	addr := flag.String("addr", ":9099", "listen address (host:port)")
	gatewayURL := flag.String("gateway-url", "", "optional A2A gateway base URL; when set, agents advertise a proxy_url (see profile-a2a-gateway.md)")
	seed := flag.Bool("seed", false, "use the in-process mock backend and install demo fixtures")
	portMin := flag.Int("port-min", 9100, "minimum agent port")
	portMax := flag.Int("port-max", 9199, "maximum agent port")
	localhostAdmin := flag.Bool("localhost-admin", true, "treat unauthenticated loopback callers as admin")
	flag.Parse()

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}

	var be backend.Backend
	var mock *backend.MockBackend
	if *seed {
		mock = backend.NewMockBackend()
		be = mock
	} else {
		be = backend.NewExecBackend()
	}

	srv := server.New(server.Config{
		GatewayBaseURL: *gatewayURL,
		PortRange:      server.PortRange{Min: *portMin, Max: *portMax},
		LocalhostAdmin: *localhostAdmin,
		Backend:        be,
	})

	if *seed {
		if err := srv.Seed(context.Background(), demoSeed()); err != nil {
			log.Fatalf("seed: %v", err)
		}
		log.Printf("seeded project=myapp workspace=arp-test agents=echo-agent-001,crush-agent-001")
	}

	gs := grpc.NewServer()
	srv.Register(gs)
	reflection.Register(gs)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		<-sigc
		log.Printf("shutting down")
		srv.Stop()
		gs.GracefulStop()
		if mock != nil {
			mock.Close()
		}
	}()

	log.Printf("ARP reference server listening on %s", lis.Addr())
	if err := gs.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func demoSeed() server.SeedConfig {
	return server.SeedConfig{
		Project: "myapp",
		Repo:    "/tmp/myapp",
		Branch:  "main",
		Templates: []*arpv1.AgentTemplate{
			{
				Name:    "crush",
				Command: "crush serve",
				PortEnv: "A2A_PORT",
				A2ACardConfig: &arpv1.A2ACardConfig{
					Name:        "Crush",
					Description: "AI coding assistant",
					Streaming:   true,
					Skills: []*arpv1.AgentSkillConfig{{
						Id:          "code",
						Name:        "Code",
						Description: "Write, review, and debug code",
						Tags:        []string{"crush", "coding", "code"},
					}},
				},
			},
			{
				Name:    "echo",
				Command: "echo serve",
				PortEnv: "A2A_PORT",
				A2ACardConfig: &arpv1.A2ACardConfig{
					Name:        "Echo",
					Description: "Echoes input",
					Skills: []*arpv1.AgentSkillConfig{{
						Id:          "echo",
						Name:        "Echo",
						Description: "Echo skill",
						Tags:        []string{"echo"},
					}},
				},
			},
		},
		Workspace: "arp-test",
		Agents: []server.SeedAgent{
			{ID: "echo-agent-001", Template: "echo", Name: "echo-agent"},
			{ID: "crush-agent-001", Template: "crush", Name: "crush-agent"},
		},
	}
}
