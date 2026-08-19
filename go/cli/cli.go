// Package main provides the MEXC command-line interface workflow.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	k4k3ruCLI "github.com/k4k3ru-hub/cli/go"
	"github.com/k4k3ru-hub/mexc/go/rest"
	spotREST "github.com/k4k3ru-hub/mexc/go/rest/spot"
)

// Option contains injectable CLI dependencies.
type Option struct {
	HTTPClient rest.HTTPClient
}

// Run executes one MEXC CLI command.
//
// Supported command:
//   - rest spot exchange-info
//
// Parameters:
//   - ctx: command context
//   - args: arguments following the executable name
//   - stdout: successful JSON output destination
//   - stderr: flag usage and parsing error destination
//   - option: injectable command dependencies
//
// Version:
//   - 2026-08-19: Added.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, option *Option) error {
	if stdout == nil {
		return fmt.Errorf("failed to run cli: stdout=null")
	}
	if stderr == nil {
		return fmt.Errorf("failed to run cli: stderr=null")
	}
	application, err := newApplication(ctx, option)
	if err != nil {
		return fmt.Errorf("failed to run cli: %w", err)
	}
	if err := application.SetIO(strings.NewReader(""), stdout, stderr); err != nil {
		return fmt.Errorf("failed to run cli: %w", err)
	}
	if err := application.RunArgs(args); err != nil {
		return fmt.Errorf("failed to run cli: %w", err)
	}
	return nil
}

func newApplication(ctx context.Context, option *Option) (*k4k3ruCLI.CLI, error) {
	application := k4k3ruCLI.NewCLIWithName("mexc", nil)
	restCommand := k4k3ruCLI.NewCommand("rest")
	restCommand.SetUsage("Execute a MEXC public REST operation.")
	spotCommand := k4k3ruCLI.NewCommand("spot")
	spotCommand.SetUsage("Execute a MEXC Spot V3 public REST operation.")
	exchangeInfoCommand := k4k3ruCLI.NewCommand("exchange-info")
	exchangeInfoCommand.SetUsage("Get MEXC Spot V3 exchange and symbol metadata.")
	if err := exchangeInfoCommand.SetArgumentCount(0, 0); err != nil {
		return nil, fmt.Errorf("failed to create exchange info command: %w", err)
	}
	definitions := []struct {
		name   string
		option k4k3ruCLI.Option
	}{
		{name: "symbol", option: k4k3ruCLI.Option{Description: "Uppercase Spot symbol, for example BTCUSDT"}},
		{name: "symbols", option: k4k3ruCLI.Option{Description: "Comma-separated uppercase Spot symbols"}},
		{name: "status", option: k4k3ruCLI.Option{Description: "MEXC symbol status filter"}},
		{name: "trade-side-type", option: k4k3ruCLI.Option{Description: "MEXC tradeSideType filter"}},
		{name: "base-url", option: k4k3ruCLI.Option{DefaultValue: spotREST.DefaultBaseURL, Description: "MEXC Spot V3 REST base URL"}},
	}
	for _, definition := range definitions {
		if err := exchangeInfoCommand.AddOption(definition.name, definition.option); err != nil {
			return nil, fmt.Errorf("failed to create exchange info command: %w: option_name=%q", err, definition.name)
		}
	}
	exchangeInfoCommand.SetAction(func(commandContext *k4k3ruCLI.Context) error {
		return runExchangeInfo(ctx, commandContext, option)
	})
	if err := spotCommand.AddCommand(exchangeInfoCommand); err != nil {
		return nil, fmt.Errorf("failed to create spot rest command: %w", err)
	}
	if err := restCommand.AddCommand(spotCommand); err != nil {
		return nil, fmt.Errorf("failed to create rest command: %w", err)
	}
	if err := application.Root().AddCommand(restCommand); err != nil {
		return nil, fmt.Errorf("failed to create cli application: %w", err)
	}
	return application, nil
}

func runExchangeInfo(ctx context.Context, commandContext *k4k3ruCLI.Context, option *Option) error {
	if commandContext == nil {
		return fmt.Errorf("failed to run exchange info command: command_context=null")
	}
	value := func(name string) string {
		parsed, ok := commandContext.Option(name)
		if !ok {
			return ""
		}
		return parsed.Value
	}
	symbols, err := parseSymbols(value("symbols"))
	if err != nil {
		return fmt.Errorf("failed to run exchange info command: %w", err)
	}
	clientOption := &rest.ClientOption{BaseURL: value("base-url")}
	if option != nil {
		clientOption.HTTPClient = option.HTTPClient
	}
	client, err := spotREST.NewClient(clientOption)
	if err != nil {
		return fmt.Errorf("failed to run exchange info command: %w", err)
	}
	result, err := client.ExchangeInfo(ctx, spotREST.ExchangeInfoParams{
		Symbol:        value("symbol"),
		Symbols:       symbols,
		Status:        value("status"),
		TradeSideType: value("trade-side-type"),
	})
	if err != nil {
		return fmt.Errorf("failed to run exchange info command: %w", err)
	}
	encoder := json.NewEncoder(commandContext.Output())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to run exchange info command: failed to encode result: %w", err)
	}
	return nil
}

func parseSymbols(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	symbols := make([]string, 0, len(parts))
	for _, part := range parts {
		symbol := strings.TrimSpace(part)
		if symbol == "" {
			return nil, fmt.Errorf("failed to parse symbols option: symbols=invalid")
		}
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}
