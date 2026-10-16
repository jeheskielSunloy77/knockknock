package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	knockknockskill "github.com/jeheskielSunloy77/knockknock/skills/knockknock"
)

var SkillContent = knockknockskill.Content

const skillUsage = `knockknock skill - manage and install AI agent skill

Usage:
  knockknock skill install [-g|--global] [--dir <path>]
  knockknock skill show
  knockknock install-skill [-g|--global] [--dir <path>]

Options:
  -g, --global       install globally to ~/.agents/skills/knockknock/SKILL.md
      --dir <path>   install to a specific directory
  -h, --help         show help and exit
`

func runSkill(cfg Config, args []string) int {
	subcommand := "install"
	var subArgs []string

	if len(args) > 0 && args[0] == "install-skill" {
		subcommand = "install"
		subArgs = args[1:]
	} else if len(args) > 0 && args[0] == "skill" {
		if len(args) == 1 {
			fmt.Fprint(cfg.Stdout, skillUsage)
			return 0
		}
		subcommand = args[1]
		subArgs = args[2:]
	}

	switch subcommand {
	case "-h", "--help", "help":
		fmt.Fprint(cfg.Stdout, skillUsage)
		return 0
	case "show", "cat", "view":
		fmt.Fprint(cfg.Stdout, SkillContent)
		return 0
	case "install":
		return runSkillInstall(cfg, subArgs)
	default:
		fmt.Fprintf(cfg.Stderr, "knockknock skill: unknown command %q (try --help)\n", subcommand)
		return 2
	}
}

func runSkillInstall(cfg Config, args []string) int {
	var isGlobal bool
	var customDir string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Fprint(cfg.Stdout, skillUsage)
			return 0
		case "-g", "--global":
			isGlobal = true
		case "--dir":
			i++
			if i >= len(args) {
				fmt.Fprintln(cfg.Stderr, "knockknock skill install: --dir requires a value")
				return 2
			}
			customDir = args[i]
		default:
			fmt.Fprintf(cfg.Stderr, "knockknock skill install: unknown option %q\n", args[i])
			return 2
		}
	}

	var targetPaths []string

	if customDir != "" {
		if strings.HasSuffix(customDir, "SKILL.md") {
			targetPaths = append(targetPaths, customDir)
		} else {
			targetPaths = append(targetPaths, filepath.Join(customDir, "SKILL.md"))
		}
	} else if isGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(cfg.Stderr, "knockknock skill install: could not determine home directory: %v\n", err)
			return 1
		}
		targetPaths = append(targetPaths, filepath.Join(home, ".agents", "skills", "knockknock", "SKILL.md"))
		if fi, err := os.Stat(filepath.Join(home, ".claude")); err == nil && fi.IsDir() {
			targetPaths = append(targetPaths, filepath.Join(home, ".claude", "skills", "knockknock", "SKILL.md"))
		}
	} else {
		targetPaths = append(targetPaths, filepath.Join(".agents", "skills", "knockknock", "SKILL.md"))
		if fi, err := os.Stat(".claude"); err == nil && fi.IsDir() {
			targetPaths = append(targetPaths, filepath.Join(".claude", "skills", "knockknock", "SKILL.md"))
		}
		if fi, err := os.Stat(".cursor"); err == nil && fi.IsDir() {
			targetPaths = append(targetPaths, filepath.Join(".cursor", "skills", "knockknock", "SKILL.md"))
		}
	}

	for _, p := range targetPaths {
		dir := filepath.Dir(p)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(cfg.Stderr, "knockknock skill install: creating directory %s: %v\n", dir, err)
			return 1
		}
		if err := os.WriteFile(p, []byte(SkillContent), 0644); err != nil {
			fmt.Fprintf(cfg.Stderr, "knockknock skill install: writing file %s: %v\n", p, err)
			return 1
		}
		fmt.Fprintf(cfg.Stdout, "installed skill to %s\n", p)
	}

	return 0
}
