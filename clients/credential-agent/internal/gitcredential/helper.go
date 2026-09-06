package gitcredential

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/boykaspaces/hermes-personal-tools/clients/credential-agent/internal/lease"
)

type Input struct {
	Protocol string
	Host     string
	Path     string
}

func Run(operation, directory string, input io.Reader, output io.Writer, now time.Time) error {
	switch operation {
	case "store", "erase":
		return nil
	case "get":
	default:
		return fmt.Errorf("unsupported git credential operation %q", operation)
	}
	query, err := parse(input)
	if err != nil {
		return err
	}
	if query.Protocol == "" || query.Host == "" || query.Path == "" {
		return errors.New("protocol, host, and path are required")
	}
	leases, err := lease.LoadDirectory(directory, now)
	if err != nil {
		return fmt.Errorf("load credential leases: %w", err)
	}
	for _, item := range leases {
		if !item.Matches(query.Protocol, query.Host, query.Path) {
			continue
		}
		_, err := fmt.Fprintf(output, "username=%s\npassword=%s\n\n", item.Credential.Username, item.Credential.Secret)
		return err
	}
	return nil
}

func parse(input io.Reader) (Input, error) {
	var result Input
	scanner := bufio.NewScanner(io.LimitReader(input, 8192))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return Input{}, errors.New("invalid git credential input")
		}
		switch key {
		case "protocol":
			result.Protocol = value
		case "host":
			result.Host = value
		case "path":
			result.Path = value
		}
	}
	if err := scanner.Err(); err != nil {
		return Input{}, err
	}
	return result, nil
}
