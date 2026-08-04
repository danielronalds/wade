package main

import (
	"fmt"
	"os"

	"wade/cmd/wade/internal/controllers"
)

// @title WADE API
// @version 0.1.0
// @description Local HTTP API for the WADE browser workspace.
// @BasePath /
// @schemes http
// @accept json
// @produce json
func main() {
	router := controllers.NewRouter(os.Stdout)
	exitCode, err := router.HandleArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
