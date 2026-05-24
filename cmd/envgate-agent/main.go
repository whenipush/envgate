package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/whenipush/envgate/gen/go/envgate/v1"
	cfg "github.com/whenipush/envgate/internal/pkg/config"
)

func main() {

	agentCfg := cfg.MustLoadConfigAgent()

	if agentCfg.Auth.Token == "" {
		log.Fatal("Critical error: ENVGATE_TOKEN environment variable is required")
	}

	log.Printf("Connecting to envgate server at %s...", agentCfg.Server.Address)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, agentCfg.Server.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("did not connect to envgate server: %v", err)
	}
	defer conn.Close()

	client := pb.NewEnvGateServiceClient(conn)

	reqCtx, reqCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer reqCancel()

	log.Println("Fetching configuration variables...")
	res, err := client.PullSecrets(reqCtx, &pb.PullSecretsRequest{
		Token: agentCfg.Auth.Token,
	})
	if err != nil {
		log.Fatalf("Failed to fetch config from server: %v", err)
	}

	log.Printf("Successfully loaded %d variables!", len(res.Variables))
	for key, value := range res.Variables {
		_ = os.Setenv(key, value)
	}

	if len(os.Args) < 2 {
		log.Fatal("Critical error: No target application command provided. Usage: envgate-agent <command> [args...]")
	}

	targetCmd := os.Args[1]
	targetArgs := os.Args[2:]

	log.Printf("Starting target application: %s %v", targetCmd, targetArgs)

	cmd := exec.Command(targetCmd, targetArgs...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		log.Fatalf("Failed to run target application: %v", err)
	}
}
