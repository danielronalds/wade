package main

import (
	"fmt"
	"os"

	"wade/cmd/wade/internal/controllers"
)

func main() {
	router := controllers.NewRouter(os.Stdout)
	if err := router.HandleArgs(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
