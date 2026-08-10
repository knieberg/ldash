package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/knieberg/ldash/internal/config"
	"github.com/knieberg/ldash/internal/tui"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ldash",
		Short: "Terminal UI for OpenLDAP administration",
		Long:  "ldash is a terminal user interface for managing OpenLDAP directories.",
		RunE:  runTUI,
	}
	root.AddCommand(newVersionCmd())
	root.AddCommand(newConfigCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration commands",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize user config from config.example.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := config.InitFromEmbedded()
			if err != nil {
				example, err2 := findExampleConfig()
				if err2 != nil {
					return err
				}
				tplExample, err3 := findExampleTemplate()
				if err3 != nil {
					return err
				}
				dest, err := config.InitFromExample(example, tplExample)
				if err != nil {
					return err
				}
				fmt.Printf("Created config at %s\n", dest)
				fmt.Println("Templates: ~/.config/ldash/templates/")
				fmt.Println("Edit the files, then run: ldash")
				return nil
			}
			createdNew := slices.Contains(res.Added, config.ConfigFile)
			if createdNew {
				fmt.Printf("Created config at %s\n", res.ConfigPath)
			} else {
				fmt.Printf("Config at %s\n", res.ConfigPath)
				fmt.Println("Added missing files:")
			}
			for _, name := range res.Added {
				fmt.Printf("  %s\n", name)
			}
			fmt.Println("Templates: ~/.config/ldash/templates/")
			fmt.Println("Edit the files, then run: ldash")
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print default config path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultConfigPath()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	})
	return cmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return cmd.Help()
	}

	cfgPath, err := config.DefaultConfigPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("config not found at %s — run: ldash config init", cfgPath)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	password, err := readBindPassword(cfg)
	if err != nil {
		return err
	}

	return tui.Run(cfg, password)
}

func readBindPassword(cfg *config.Config) (string, error) {
	credPath, err := cfg.CredentialPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(credPath); err == nil {
		return config.ReadBindPasswordFile(credPath)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "LDAP bind password for %s: ", cfg.BindDN)
	pass, err := readPasswordTerminal()
	if err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr)
	return pass, nil
}

func readPasswordTerminal() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("stdin is not a terminal; set credential_file in config")
	}
	b, err := term.ReadPassword(fd)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func findExampleConfig() (string, error) {
	return findProjectFile("config.example.yaml")
}

func findExampleTemplate() (string, error) {
	return findProjectFile(filepath.Join("internal", "templates", "user_samba_posix.example.yaml"))
}

func findProjectFile(rel string) (string, error) {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "..", rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(cwd, rel)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("%s not found; run from project root", rel)
}
