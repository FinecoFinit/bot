package net

//import (
//	"context"
//	"time"
//
//	goipam "github.com/metal-stack/go-ipam"
//)
//
//func NewIPCTX() {
//	// The background context
//	bgCtx := context.Background()
//
//	// Create a ipamer with in memory storage
//	ipam := goipam.New(bgCtx)
//
//	// Optionally, we can pass around a context for a given namespace
//	namespace := "tenant-a"
//	err := ipam.CreateNamespace(bgCtx, namespace)
//	if err != nil {
//		panic(err)
//	}
//	ctx := goipam.NewContextWithNamespace(bgCtx, namespace)
//	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
//	defer cancel()
//}
