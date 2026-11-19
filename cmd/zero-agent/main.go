// Command zero-agent is the host-side service that discovers and supervises
// protocol plugins, reports them to zero-web-server, and proxies plugin RPC.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/yaml.v3"

	"zero-web-server/internal/agent"
)

func main() {
	configPath := flag.String("config", "configs/agent.yaml", "agent config file")
	flag.Parse()

	cfg := agent.Config{
		Listen:       "0.0.0.0:19000",
		PluginDir:    "plugins",
		StateFile:    "data/state.json",
		Heartbeat:    "10s",
		StartTimeout: "15s",
	}
	raw, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("read config %s: %v", *configPath, err)
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("parse config %s: %v", *configPath, err)
	}

	a, err := agent.New(cfg)
	if err != nil {
		log.Fatalf("init agent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()
	if err := a.Run(ctx); err != nil {
		log.Fatalf("agent exit: %v", err)
	}
}
