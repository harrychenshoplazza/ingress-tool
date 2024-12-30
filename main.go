package main

import (
	"ingress-tool/api"
	"log"
	"os"
)

func main() {
	kubeConfig := os.Getenv("HOME") + "/.kube/config"
	clienset, err := api.InitK8sClient(kubeConfig)
	if err != nil {
		log.Fatalf("Fail to initialize Kubernetes client: %v", err)
	}
	router := api.SetupRouter(clienset)
	log.Println("Server running at http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Fail to run server: %v", err)
	}
}
