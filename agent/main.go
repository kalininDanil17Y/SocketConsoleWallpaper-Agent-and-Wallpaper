package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kardianos/service"

	"socket-console-agent/internal/api"
	"socket-console-agent/internal/config"
)

const (
	serviceName        = "SocketConsoleAgent"
	serviceDisplayName = "Socket Console Agent"
	serviceDescription = "Localhost system metrics and ASCII image bridge for Wallpaper Engine."
)

type program struct {
	logger service.Logger
	server *api.Server
	cancel context.CancelFunc
}

var serviceConfigPathOverride string

func main() {
	serviceConfigPathOverride = parseServiceConfigPathArg(os.Args[1:])

	svc, err := newAgentService(nil)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 && service.Interactive() {
		hideConsoleForUI()
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
	}

	if len(os.Args) >= 2 && os.Args[1] == "service" {
		if err := svc.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(os.Args) < 2 {
		if service.Interactive() {
			if err := runUI(svc); err != nil {
				log.Fatal(err)
			}
			return
		}
		if err := svc.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(os.Args) >= 3 && isHelpArg(os.Args[2]) {
		if !printCommandHelp(os.Args[1]) {
			fmt.Printf("Unknown command: %s\n\n", os.Args[1])
			printGeneralHelp()
			os.Exit(2)
		}
		return
	}

	switch os.Args[1] {
	case "help", "--help", "-h", "/?":
		if len(os.Args) >= 3 {
			if !printCommandHelp(os.Args[2]) {
				fmt.Printf("Unknown command: %s\n\n", os.Args[2])
				printGeneralHelp()
				os.Exit(2)
			}
			return
		}
		printGeneralHelp()
	case "ui":
		hideConsoleForUI()
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
		if err := runUI(svc); err != nil {
			log.Fatal(err)
		}
	case "run":
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
		if _, installed, _ := serviceState(svc); installed {
			log.Fatal("service is installed; use start/stop/status or uninstall it before local run")
		}
		if err := runConsole(); err != nil {
			log.Fatal(err)
		}
	case "install":
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
		if err := svc.Install(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Service installed.")
	case "uninstall":
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
		if err := svc.Uninstall(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Service uninstalled.")
	case "start":
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
		if err := svc.Start(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Service started.")
	case "stop":
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
		if err := svc.Stop(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Service stopped.")
	case "status":
		relaunch, err := ensureElevated()
		if err != nil {
			log.Fatal(err)
		}
		if relaunch {
			return
		}
		status, err := svc.Status()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(serviceStatusText(status))
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printGeneralHelp()
		os.Exit(2)
	}
}

func newAgentService(arguments []string) (service.Service, error) {
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		Arguments:   arguments,
		Option: service.KeyValue{
			"StartType": "automatic",
		},
	}

	prg := &program{}
	return service.New(prg, svcConfig)
}

func runConsole() error {
	cfg, cfgPath, err := config.Load(config.DevConfigPath())
	if err != nil {
		return err
	}

	fmt.Printf("Using config: %s\n", cfgPath)
	srv := api.NewServer(cfg, cfgPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	fmt.Printf("Socket Console Agent listening on http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("Press Ctrl+C to stop.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func (p *program) Start(s service.Service) error {
	logger, err := s.Logger(nil)
	if err == nil {
		p.logger = logger
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	cfg, cfgPath, err := config.Load(serviceConfigPath())
	if err != nil {
		if p.logger != nil {
			_ = p.logger.Error(err)
		}
		return err
	}

	p.server = api.NewServer(cfg, cfgPath)
	go func() {
		if err := p.server.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			if p.logger != nil {
				_ = p.logger.Error(err)
			}
		}
	}()

	if p.logger != nil {
		_ = p.logger.Infof("Socket Console Agent started on %s:%d", cfg.Server.Host, cfg.Server.Port)
	}
	return nil
}

func (p *program) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

func printGeneralHelp() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`Socket Console Agent

Localhost Windows metrics and color ASCII image bridge for Wallpaper Engine.

Usage:
  %s <command>
  %s help <command>

Commands:
  ui         Open the native Windows control panel.
  run        Run the agent in the current console for development.
  install    Install the agent as a Windows Service with automatic startup.
  uninstall  Remove the Windows Service registration.
  start      Start the installed Windows Service.
  stop       Stop the installed Windows Service.
  status     Print the current Windows Service status.
  help       Show general help or command-specific help.

HTTP endpoints:
  GET  http://127.0.0.1:48771/api/v1/status
  GET  http://127.0.0.1:48771/api/v1/config
  POST http://127.0.0.1:48771/api/v1/config
  GET  http://127.0.0.1:48771/api/v1/interfaces
  GET  http://127.0.0.1:48771/api/v1/disks
  GET  http://127.0.0.1:48771/api/v1/images
  GET  http://127.0.0.1:48771/api/v1/ascii
  WS   ws://127.0.0.1:48771/api/v1/live

Config:
  Dev mode:        .\config.json
  Service mode:    %%ProgramData%%\SocketConsoleAgent\config.json
  Override:        SOCKET_CONSOLE_AGENT_CONFIG=C:\path\to\config.json

Examples:
  %s ui
  %s run
  %s help run
  %s install
`, exe, exe, exe, exe, exe, exe)
}

func printCommandHelp(command string) bool {
	exe := filepath.Base(os.Args[0])
	helps := map[string]string{
		"ui": fmt.Sprintf(`Command: ui

Usage:
  %s ui

Opens the native Windows control panel. The window can run the agent locally
when the service is not installed, manage the Windows Service, show logs, and
edit the service config.

Example:
  %s ui
`, exe, exe),
		"run": fmt.Sprintf(`Command: run

Usage:
  %s run

Starts the agent in the current console. Use this mode while developing or testing
Wallpaper Engine integration.

Behavior:
  - reads .\config.json by default
  - creates the default config if it does not exist
  - listens only on 127.0.0.1
  - stops cleanly on Ctrl+C

Example:
  %s run
`, exe, exe),
		"install": fmt.Sprintf(`Command: install

Usage:
  %s install

Installs Socket Console Agent as a Windows Service and configures automatic
startup. Run this from an elevated terminal.

Service config path:
  %%ProgramData%%\SocketConsoleAgent\config.json

Example:
  %s install
`, exe, exe),
		"uninstall": fmt.Sprintf(`Command: uninstall

Usage:
  %s uninstall

Removes the Windows Service registration. Stop the service first if it is
currently running. Run this from an elevated terminal.

Example:
  %s stop
  %s uninstall
`, exe, exe, exe),
		"start": fmt.Sprintf(`Command: start

Usage:
  %s start

Starts the installed Windows Service. The service must be installed first.

Example:
  %s install
  %s start
`, exe, exe, exe),
		"stop": fmt.Sprintf(`Command: stop

Usage:
  %s stop

Stops the installed Windows Service.

Example:
  %s stop
`, exe, exe),
		"status": fmt.Sprintf(`Command: status

Usage:
  %s status

Prints the Windows Service status.

Possible output:
  running
  stopped
  unknown (<code>)

Example:
  %s status
`, exe, exe),
		"help": fmt.Sprintf(`Command: help

Usage:
  %s help
  %s help <command>

Shows general help or detailed help for a specific command.

Examples:
  %s help
  %s help run
  %s help install
`, exe, exe, exe, exe, exe),
	}

	text, ok := helps[command]
	if !ok {
		return false
	}
	fmt.Print(text)
	return true
}

func serviceStatusText(status service.Status) string {
	switch status {
	case service.StatusRunning:
		return "running"
	case service.StatusStopped:
		return "stopped"
	default:
		return fmt.Sprintf("unknown (%d)", status)
	}
}

func isHelpArg(arg string) bool {
	switch arg {
	case "help", "--help", "-h", "/?":
		return true
	default:
		return false
	}
}

func serviceConfigPath() string {
	if serviceConfigPathOverride != "" {
		return serviceConfigPathOverride
	}
	return config.ServiceConfigPath()
}

func parseServiceConfigPathArg(args []string) string {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config", "-config":
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}

func serviceArgumentsForConfigPath(path string) []string {
	if path == "" {
		return []string{"service"}
	}
	return []string{"service", "--config", path}
}
