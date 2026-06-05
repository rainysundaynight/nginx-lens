package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/rainysundaynight/nginx-lens/internal/config"
	"github.com/rainysundaynight/nginx-lens/internal/hub"
)

func main() {
	cfg := config.Get().Config.Web.Hub
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("nginx-lens-hub listening on %s", addr)
	if err := http.ListenAndServe(addr, hub.NewRouter()); err != nil {
		log.Fatal(err)
	}
}
