package main

import (
	"context"
	"fmt"
	"os"

	"github.com/apparentlymart/terraform-provider/tfprovider"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <provider-executable> [provider-args...]\n", args[0])
		os.Exit(1)
	}
	args = args[1:]

	ctx := context.Background()
	provider, err := tfprovider.Start(ctx, args[0], args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	defer provider.Close()

	schema, diags := provider.Schema(ctx)
	showDiagnosticsMaybeExit(diags, provider)

	if len(schema.ManagedResourceTypes) != 0 {
		fmt.Print("\n# Managed Resource Types\n\n")
		for name := range schema.ManagedResourceTypes {
			fmt.Printf("- %s\n", name)
		}
	}

	if len(schema.DataResourceTypes) != 0 {
		fmt.Print("\n# Data Resource Types\n\n")
		for name := range schema.DataResourceTypes {
			fmt.Printf("- %s\n", name)
		}
	}

	fmt.Print("\n")
}

func showDiagnosticsMaybeExit(diags tfprovider.Diagnostics, provider tfprovider.Provider) {
	for _, diag := range diags {
		switch diag.Severity {
		case tfprovider.Error:
			fmt.Fprintf(os.Stderr, "Error: %s; %s", diag.Summary, diag.Detail)
		case tfprovider.Warning:
			fmt.Fprintf(os.Stderr, "Warning: %s; %s", diag.Summary, diag.Detail)
		default:
			fmt.Fprintf(os.Stderr, "???: %s; %s", diag.Summary, diag.Detail)
		}
	}
	if diags.HasErrors() {
		provider.Close()
		os.Exit(1)
	}
}
