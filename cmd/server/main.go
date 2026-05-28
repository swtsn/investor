package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"google.golang.org/grpc"

	investorv1 "github.com/swtsn/investor/gen/investor/v1"
	"github.com/swtsn/investor/internal/db"
	investorgrpc "github.com/swtsn/investor/internal/grpc"
	"github.com/swtsn/investor/internal/service"
)

func main() {
	dbPath := os.Getenv("INVESTOR_DB")
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("home dir: %v", err)
		}
		dbPath = filepath.Join(home, ".investor", "investor.db")
	}

	addr := os.Getenv("INVESTOR_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	store, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget := service.NewBudgetService(store)
	deploy := service.NewDeploymentService(store)
	pool := service.NewPoolService(store)

	handler := investorgrpc.NewInvestorHandler(budget, deploy, pool)

	srv := grpc.NewServer()
	investorv1.RegisterInvestorServiceServer(srv, handler)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	fmt.Printf("investor-server listening on %s\n", addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
