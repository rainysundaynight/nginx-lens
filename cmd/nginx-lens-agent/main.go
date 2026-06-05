package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/rainysundaynight/nginx-lens/internal/agent"
	"github.com/rainysundaynight/nginx-lens/internal/config"
)

func main() {
	cfg := config.Get().Config.Web.Agent
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("nginx-lens-agent listening on %s", addr)
	if err := http.ListenAndServe(addr, agent.NewRouter()); err != nil {
		log.Fatal(err)
	}
}
