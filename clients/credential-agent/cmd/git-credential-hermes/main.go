package main

import (
	"fmt"
	"os"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/clients/credential-agent/internal/gitcredential"
)

const defaultCredentialDirectory = "/run/hermes/credentials"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: git-credential-hermes <get|store|erase>")
		os.Exit(2)
	}
	directory := os.Getenv("HERMES_CREDENTIAL_DIR")
	if directory == "" {
		directory = defaultCredentialDirectory
	}
	if err := gitcredential.Run(os.Args[1], directory, os.Stdin, os.Stdout, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "git-credential-hermes: credential unavailable")
		os.Exit(1)
	}
}
