//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("Starting development server with hot reload...")

	// Install air if not present
	cmd := exec.Command("go", "install", "github.com/air-verse/air@latest")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to install air: %v\n", err)
		fmt.Println("Falling back to regular go run...")
		fallbackCmd := exec.Command("go", "run", "cmd/server/main.go")
		fallbackCmd.Stdout = os.Stdout
		fallbackCmd.Stderr = os.Stderr
		fallbackCmd.Stdin = os.Stdin
		if err := fallbackCmd.Run(); err != nil {
			fmt.Printf("Error starting server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Run with air using our config
	airCmd := exec.Command("air", "-c", "config/air.toml")
	airCmd.Stdout = os.Stdout
	airCmd.Stderr = os.Stderr
	airCmd.Stdin = os.Stdin

	err := airCmd.Run()
	if err != nil {
		fmt.Printf("Error starting development server: %v\n", err)
		os.Exit(1)
	}
}
