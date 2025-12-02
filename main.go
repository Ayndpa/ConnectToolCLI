package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Define flags
	socketPath := flag.String("socket", defaultSocketPath(), "Path to the Unix Domain Socket")
	flag.Parse()

	if len(flag.Args()) < 1 {
		printUsage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	// Connect to gRPC server
	// Note: On Windows, we might need "unix:" prefix explicitly if it's not handled by the dialer target parser correctly for relative paths,
	// but generally "unix:path" works.
	target := "unix:" + *socketPath
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := NewConnectToolServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch command {
	case "create":
		createLobby(ctx, client)
	case "join":
		if len(flag.Args()) < 2 {
			log.Fatal("Usage: join <lobby_id>")
		}
		joinLobby(ctx, client, flag.Arg(1))
	case "leave":
		leaveLobby(ctx, client)
	case "info":
		getLobbyInfo(ctx, client)
	case "friends":
		getFriendLobbies(ctx, client)
	case "invite":
		if len(flag.Args()) < 2 {
			log.Fatal("Usage: invite <steam_id>")
		}
		inviteFriend(ctx, client, flag.Arg(1))

	case "vpn-status":
		getVPNStatus(ctx, client)
	case "vpn-routes":
		getVPNRoutingTable(ctx, client)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func defaultSocketPath() string {
	if runtime.GOOS == "windows" {
		return "connect_tool.sock"
	}
	return "/tmp/connect_tool.sock"
}

func printUsage() {
	fmt.Println("Usage: connecttoolcli [flags] <command> [args...]")
	fmt.Println("Commands:")
	fmt.Println("  create                   Create a new lobby")
	fmt.Println("  join <lobby_id>          Join a lobby")
	fmt.Println("  leave                    Leave current lobby")
	fmt.Println("  info                     Get current lobby info")
	fmt.Println("  friends                  List friend lobbies")
	fmt.Println("  invite <steam_id>        Invite a friend")

	fmt.Println("  vpn-status               Get VPN status")
	fmt.Println("  vpn-routes               Get VPN routing table")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func createLobby(ctx context.Context, client ConnectToolServiceClient) {
	r, err := client.CreateLobby(ctx, &CreateLobbyRequest{})
	if err != nil {
		log.Fatalf("could not create lobby: %v", err)
	}
	fmt.Printf("Success: %v, Lobby ID: %s\n", r.GetSuccess(), r.GetLobbyId())
}

func joinLobby(ctx context.Context, client ConnectToolServiceClient, lobbyID string) {
	r, err := client.JoinLobby(ctx, &JoinLobbyRequest{LobbyId: lobbyID})
	if err != nil {
		log.Fatalf("could not join lobby: %v", err)
	}
	fmt.Printf("Success: %v, Message: %s\n", r.GetSuccess(), r.GetMessage())
}

func leaveLobby(ctx context.Context, client ConnectToolServiceClient) {
	r, err := client.LeaveLobby(ctx, &LeaveLobbyRequest{})
	if err != nil {
		log.Fatalf("could not leave lobby: %v", err)
	}
	fmt.Printf("Success: %v\n", r.GetSuccess())
}

func getLobbyInfo(ctx context.Context, client ConnectToolServiceClient) {
	r, err := client.GetLobbyInfo(ctx, &GetLobbyInfoRequest{})
	if err != nil {
		log.Fatalf("could not get lobby info: %v", err)
	}
	fmt.Printf("In Lobby: %v\n", r.GetIsInLobby())
	if r.GetIsInLobby() {
		fmt.Printf("Lobby ID: %s\n", r.GetLobbyId())
		fmt.Println("Members:")
		for _, m := range r.GetMembers() {
			fmt.Printf("  - Name: %s, ID: %s, Ping: %d, Relay: %s\n", m.GetName(), m.GetSteamId(), m.GetPing(), m.GetRelayInfo())
		}
	}
}

func getFriendLobbies(ctx context.Context, client ConnectToolServiceClient) {
	r, err := client.GetFriendLobbies(ctx, &GetFriendLobbiesRequest{})
	if err != nil {
		log.Fatalf("could not get friend lobbies: %v", err)
	}
	fmt.Println("Friend Lobbies:")
	for _, l := range r.GetLobbies() {
		fmt.Printf("  - Friend: %s (%s), Lobby: %s\n", l.GetName(), l.GetSteamId(), l.GetLobbyId())
	}
}

func inviteFriend(ctx context.Context, client ConnectToolServiceClient, friendID string) {
	r, err := client.InviteFriend(ctx, &InviteFriendRequest{FriendSteamId: friendID})
	if err != nil {
		log.Fatalf("could not invite friend: %v", err)
	}
	fmt.Printf("Success: %v\n", r.GetSuccess())
}

func getVPNStatus(ctx context.Context, client ConnectToolServiceClient) {
	r, err := client.GetVPNStatus(ctx, &GetVPNStatusRequest{})
	if err != nil {
		log.Fatalf("could not get VPN status: %v", err)
	}
	fmt.Printf("Enabled: %v\n", r.GetEnabled())
	if r.GetEnabled() {
		fmt.Printf("Local IP: %s\n", r.GetLocalIp())
		fmt.Printf("Device: %s\n", r.GetDeviceName())
		stats := r.GetStats()
		if stats != nil {
			fmt.Println("Stats:")
			fmt.Printf("  Sent: %d pkts / %d bytes\n", stats.GetPacketsSent(), stats.GetBytesSent())
			fmt.Printf("  Recv: %d pkts / %d bytes\n", stats.GetPacketsReceived(), stats.GetBytesReceived())
			fmt.Printf("  Dropped: %d pkts\n", stats.GetPacketsDropped())
		}
	}
}

func getVPNRoutingTable(ctx context.Context, client ConnectToolServiceClient) {
	r, err := client.GetVPNRoutingTable(ctx, &GetVPNRoutingTableRequest{})
	if err != nil {
		log.Fatalf("could not get VPN routing table: %v", err)
	}
	fmt.Println("Routing Table:")
	for _, route := range r.GetRoutes() {
		// Convert uint32 IP to string
		ip := fmt.Sprintf("%d.%d.%d.%d", byte(route.GetIp()>>24), byte(route.GetIp()>>16), byte(route.GetIp()>>8), byte(route.GetIp()))
		fmt.Printf("  - IP: %s, Name: %s, Local: %v\n", ip, route.GetName(), route.GetIsLocal())
	}
}
