package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"google.golang.org/grpc"

	investorv1 "github.com/swtsn/investor/gen/investor/v1"
	"github.com/swtsn/investor/internal/db"
	investorgrpc "github.com/swtsn/investor/internal/grpc"
	"github.com/swtsn/investor/internal/service"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("home dir: %v", err)
	}
	var cli struct {
		DB   string `env:"INVESTOR_DB"   default:"${home}/.investor/investor.db" help:"Path to SQLite database"`
		Addr string `env:"INVESTOR_ADDR" default:"localhost:50051"               help:"Listen address"`
	}
	kong.Parse(&cli, kong.Vars{"home": home})

	if err := os.MkdirAll(filepath.Dir(cli.DB), 0o700); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	store, err := db.Open(cli.DB)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget := service.NewBudgetService(store)
	deploy := service.NewDeploymentService(store)
	pool := service.NewPoolService(store)
	bucket := service.NewBucketService(store)

	handler := investorgrpc.NewInvestorHandler(budget, deploy, pool, bucket)

	srv := grpc.NewServer()
	investorv1.RegisterInvestorServiceServer(srv, handler)

	lis, err := net.Listen("tcp", cli.Addr)
	if err != nil {
		log.Fatalf("listen %s: %v", cli.Addr, err)
	}
	fmt.Printf("investor-server listening on %s\n", cli.Addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
