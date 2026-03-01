package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	Endpoint string
	Token    string
	Username string
	Password string
	Wallets  []string
	Insecure bool
}

type walletsFlag []string

func (w *walletsFlag) String() string { return strings.Join(*w, ",") }
func (w *walletsFlag) Set(value string) error {
	*w = append(*w, value)
	return nil
}

func Parse() *Config {
	cfg := &Config{}
	var wallets walletsFlag

	flag.StringVar(&cfg.Endpoint, "endpoint", "", "Yellowstone gRPC endpoint URL (e.g. https://api.rpcpool.com)")
	flag.StringVar(&cfg.Token, "token", "", "Authentication token (x-token)")
	flag.StringVar(&cfg.Username, "username", "", "Basic auth username")
	flag.StringVar(&cfg.Password, "password", "", "Basic auth password")
	flag.BoolVar(&cfg.Insecure, "insecure", false, "Use insecure (non-TLS) connection")
	flag.Var(&wallets, "wallet", "Wallet address to monitor (repeatable)")
	flag.Parse()

	cfg.Wallets = wallets
	return cfg
}

func (c *Config) HasBasicAuth() bool {
	return c.Username != "" && c.Password != ""
}

func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("--endpoint is required")
	}
	if _, err := url.Parse(c.Endpoint); err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	if len(c.Wallets) == 0 {
		return fmt.Errorf("at least one --wallet is required")
	}
	return nil
}
