package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/aleksclark/spec-torture/internal/report"
	"github.com/aleksclark/spec-torture/internal/runner"
	"github.com/aleksclark/spec-torture/internal/schema"
	"github.com/aleksclark/spec-torture/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	dbPath string
	logger *slog.Logger
)

func main() {
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	root := &cobra.Command{
		Use:   "spec-torture",
		Short: "Torture-test agent runtimes against protocol specifications",
		Long:  "A test harness that runs protocol conformance tests against agent runtimes in Docker containers.",
	}

	root.PersistentFlags().StringVar(&dbPath, "db", "spec-torture.db", "path to SQLite database")

	root.AddCommand(runCmd())
	root.AddCommand(listCmd())
	root.AddCommand(reportCmd())
	root.AddCommand(validateCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func openStore() (*store.Store, error) {
	return store.New(dbPath)
}

func runCmd() *cobra.Command {
	var (
		runtimeName    string
		runtimeVersion string
		image          string
		baseURL        string
		rpcPath        string
		tags           []string
		format         string
	)

	cmd := &cobra.Command{
		Use:   "run <spec-file>",
		Short: "Run a spec against a runtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			specFile := args[0]

			if image == "" && baseURL == "" {
				return fmt.Errorf("either --image or --url must be provided")
			}

			spec, err := loadSpec(specFile)
			if err != nil {
				return fmt.Errorf("loading spec: %w", err)
			}

			st, err := openStore()
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer st.Close()

			rawYAML, _ := os.ReadFile(specFile)
			if err := st.SaveSpec(spec, string(rawYAML)); err != nil {
				return fmt.Errorf("saving spec: %w", err)
			}

			cfg := runner.Config{
				Runtime:        runtimeName,
				RuntimeVersion: runtimeVersion,
				Image:          image,
				BaseURL:        baseURL,
				RPCPath:        rpcPath,
				Tags:           tags,
			}

			r, err := runner.New(logger, cfg)
			if err != nil {
				return fmt.Errorf("creating runner: %w", err)
			}
			defer r.Close()

			result, err := r.Run(cmd.Context(), spec, cfg)
			if err != nil {
				return fmt.Errorf("running spec: %w", err)
			}

			if err := st.SaveTestRun(result); err != nil {
				return fmt.Errorf("saving results: %w", err)
			}

			f := report.FormatMarkdown
			if format == "json" {
				f = report.FormatJSON
			}

			return report.Write(os.Stdout, result, f)
		},
	}

	cmd.Flags().StringVar(&runtimeName, "runtime", "", "runtime identifier (e.g., 'claude-code-v1.2')")
	cmd.Flags().StringVar(&runtimeVersion, "runtime-version", "", "runtime version")
	cmd.Flags().StringVar(&image, "image", "", "Docker image to test against")
	cmd.Flags().StringVar(&baseURL, "url", "", "base URL of the runtime under test (for http-rest or jsonrpc-http transport)")
	cmd.Flags().StringVar(&rpcPath, "rpc-path", "", "subpath for JSON-RPC POST requests (e.g., /invoke, /jsonrpc)")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "filter test cases by tags")
	cmd.Flags().StringVar(&format, "format", "markdown", "output format (markdown, json)")
	_ = cmd.MarkFlagRequired("runtime")

	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available specs",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			specs, err := st.ListSpecs()
			if err != nil {
				return err
			}

			if len(specs) == 0 {
				fmt.Println("No specs loaded. Use 'spec-torture run <spec-file>' to load and run a spec.")
				return nil
			}

			fmt.Printf("%-20s %-30s %-10s %s\n", "ID", "NAME", "VERSION", "TRANSPORT")
			fmt.Printf("%-20s %-30s %-10s %s\n", "----", "----", "-------", "---------")
			for _, s := range specs {
				fmt.Printf("%-20s %-30s %-10s %s\n", s.ID, s.Name, s.Version, s.Transport)
			}
			return nil
		},
	}
}

func reportCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "report <run-id>",
		Short: "Show results of a test run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			run, err := st.GetTestRun(args[0])
			if err != nil {
				return fmt.Errorf("getting test run: %w", err)
			}

			f := report.FormatMarkdown
			if format == "json" {
				f = report.FormatJSON
			}

			return report.Write(os.Stdout, run, f)
		},
	}

	cmd.Flags().StringVar(&format, "format", "markdown", "output format (markdown, json)")
	return cmd
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <spec-file>",
		Short: "Validate a spec YAML file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := loadSpec(args[0])
			if err != nil {
				return err
			}

			errs := spec.Validate()
			if len(errs) == 0 {
				fmt.Printf("Spec %q is valid (%d test cases)\n", spec.ID, len(spec.TestCases))
				return nil
			}

			fmt.Fprintf(os.Stderr, "Spec %q has %d validation error(s):\n", spec.ID, len(errs))
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
			return fmt.Errorf("validation failed")
		},
	}
}

func loadSpec(path string) (*schema.Spec, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading spec file: %w", err)
	}

	var spec schema.Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec YAML: %w", err)
	}

	return &spec, nil
}
