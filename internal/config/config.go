// Package config resolves trello configuration from flags, environment
// variables, project-local .trello.yaml files (walking up to $HOME, per-key
// nearest-wins), and the user-level config file at
// <UserConfigDir>/trello/config.yaml.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/nomadicworks/trello-cli/internal/output"
)

const (
	projectFileName  = ".trello.yaml"
	projectFileAlias = ".trello.yml"
	globalFileName   = "config.yaml"
)

// Credentials are the values supplied by command-line flags.
type Credentials struct {
	Board string
	Key   string
	Token string
}

// Env are the credential values supplied by the environment.
type Env struct {
	APIKey string
	Token  string
}

// Value is a resolved setting together with the source it came from.
type Value struct {
	Present bool
	Value   string
	Source  string // "flag", "environment", or a config file path
}

// Resolved is the fully resolved view of the configuration. Credential
// values are kept private to this package's consumers; only presence and
// source are exposed through ShowView.
type Resolved struct {
	Board Value
	Key   Value
	Token Value
}

// Error is a configuration error (exit code 3).
type Error struct {
	Err error
}

func (e *Error) Error() string { return e.Err.Error() }

// Unwrap exposes the underlying cause.
func (e *Error) Unwrap() error { return e.Err }

// ExitCode reports the config-error exit code.
func (e *Error) ExitCode() int { return output.ExitConfig }

func configError(err error) error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return err
	}
	return &Error{Err: err}
}

// Resolver discovers and merges configuration files.
type Resolver struct {
	StartDir  string // walk-up start (typically the CWD)
	HomeDir   string // inclusive walk-up stop
	GlobalDir string // user config directory (<UserConfigDir>/trello)
	Warn      func(string)
}

// NewResolver returns a Resolver. warn, when non-nil, receives unknown-key
// warnings.
func NewResolver(startDir, homeDir, globalDir string, warn func(string)) *Resolver {
	return &Resolver{StartDir: startDir, HomeDir: homeDir, GlobalDir: globalDir, Warn: warn}
}

// fileData is the parsed content of a single config file.
type fileData struct {
	Path  string
	Board string
	Key   string
	Token string
}

// parseConfigFile reads and validates a YAML config file. Unknown top-level
// keys produce a warning; known keys must be scalar values.
func parseConfigFile(path string, warn func(string)) (fileData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fileData{}, configError(fmt.Errorf("%s: %w", path, err))
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fileData{}, configError(fmt.Errorf("%s: invalid YAML: %w", path, err))
	}
	fd := fileData{Path: path}
	if len(doc.Content) == 0 {
		return fd, nil // empty file
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fileData{}, configError(fmt.Errorf("%s: expected a YAML mapping of keys (board/key/token)", path))
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		keyNode := root.Content[i]
		valNode := root.Content[i+1]
		switch keyNode.Value {
		case "board", "key", "token":
			if valNode.Kind != yaml.ScalarNode {
				return fileData{}, configError(fmt.Errorf("%s: key %q must be a scalar value", path, keyNode.Value))
			}
			switch keyNode.Value {
			case "board":
				fd.Board = valNode.Value
			case "key":
				fd.Key = valNode.Value
			case "token":
				fd.Token = valNode.Value
			}
		default:
			if warn != nil {
				warn(fmt.Sprintf("%s: unknown key %q (expected board, key, or token)", path, keyNode.Value))
			}
		}
	}
	return fd, nil
}

// projectFiles parses every project-local config file from StartDir upward to
// HomeDir (inclusive), nearest first.
func (r *Resolver) projectFiles() ([]fileData, error) {
	var files []fileData
	dir := r.StartDir
	for {
		for _, name := range []string{projectFileName, projectFileAlias} {
			p := filepath.Join(dir, name)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				fd, err := parseConfigFile(p, r.Warn)
				if err != nil {
					return nil, err
				}
				files = append(files, fd)
				break
			}
		}
		if dir == r.HomeDir || dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return files, nil
}

// Resolve merges flags, env, project-local config, and global config into a
// single view.
//
// Credential precedence: flag → env → project config (nearest-wins) → global.
// Board precedence: flag → project config (nearest-wins) → global. There is
// no board environment variable.
func (r *Resolver) Resolve(flags Credentials, env Env) (Resolved, error) {
	files, err := r.projectFiles()
	if err != nil {
		return Resolved{}, err
	}

	var global fileData
	if r.GlobalDir != "" {
		gp := filepath.Join(r.GlobalDir, globalFileName)
		if fi, err := os.Stat(gp); err == nil && !fi.IsDir() {
			gd, err := parseConfigFile(gp, r.Warn)
			if err != nil {
				return Resolved{}, err
			}
			global = gd
		}
	}

	res := Resolved{}

	switch {
	case flags.Key != "":
		res.Key = Value{Present: true, Value: flags.Key, Source: "flag"}
	case env.APIKey != "":
		res.Key = Value{Present: true, Value: env.APIKey, Source: "environment"}
	default:
		for _, f := range files {
			if f.Key != "" {
				res.Key = Value{Present: true, Value: f.Key, Source: f.Path}
				break
			}
		}
		if !res.Key.Present && global.Key != "" {
			res.Key = Value{Present: true, Value: global.Key, Source: global.Path}
		}
	}

	switch {
	case flags.Token != "":
		res.Token = Value{Present: true, Value: flags.Token, Source: "flag"}
	case env.Token != "":
		res.Token = Value{Present: true, Value: env.Token, Source: "environment"}
	default:
		for _, f := range files {
			if f.Token != "" {
				res.Token = Value{Present: true, Value: f.Token, Source: f.Path}
				break
			}
		}
		if !res.Token.Present && global.Token != "" {
			res.Token = Value{Present: true, Value: global.Token, Source: global.Path}
		}
	}

	switch {
	case flags.Board != "":
		res.Board = Value{Present: true, Value: flags.Board, Source: "flag"}
	default:
		for _, f := range files {
			if f.Board != "" {
				res.Board = Value{Present: true, Value: f.Board, Source: f.Path}
				break
			}
		}
		if !res.Board.Present && global.Board != "" {
			res.Board = Value{Present: true, Value: global.Board, Source: global.Path}
		}
	}

	return res, nil
}

// Paths returns the project config path (the nearest existing file found by
// walking up, or the would-be path in StartDir) and the global config path.
func (r *Resolver) Paths() (project, global string) {
	project = filepath.Join(r.StartDir, projectFileName)
	dir := r.StartDir
	for {
		for _, name := range []string{projectFileName, projectFileAlias} {
			p := filepath.Join(dir, name)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p, filepath.Join(r.GlobalDir, globalFileName)
			}
		}
		if dir == r.HomeDir || dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return project, filepath.Join(r.GlobalDir, globalFileName)
}

// InitInput is the data written by Init.
type InitInput struct {
	Board string
	Key   string // optional; written only when non-empty
	Token string // optional; written only when non-empty
}

// Init creates or updates StartDir/.trello.yaml with the given board (and
// optional key/token). An existing file is merged in place, preserving
// unknown keys. The write is atomic and the file is created with 0600
// permissions.
func (r *Resolver) Init(in InitInput) (string, error) {
	if in.Board == "" {
		return "", configError(errors.New("no board specified; pass --board or select one interactively"))
	}
	path := filepath.Join(r.StartDir, projectFileName)

	values := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(raw, &values); err != nil {
			return "", configError(fmt.Errorf("%s: invalid YAML: %w", path, err))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", configError(fmt.Errorf("%s: %w", path, err))
	}

	values["board"] = in.Board
	if in.Key != "" {
		values["key"] = in.Key
	}
	if in.Token != "" {
		values["token"] = in.Token
	}

	out, err := yaml.Marshal(values)
	if err != nil {
		return "", configError(err)
	}
	if err := writeFileAtomic(path, out); err != nil {
		return "", configError(err)
	}
	return path, nil
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if rmErr := os.Remove(tmp); rmErr != nil {
			return fmt.Errorf("rename %s: %v (also failed to remove %s: %v)", path, err, tmp, rmErr)
		}
		return err
	}
	return nil
}

// BoardView is the JSON-safe view of the resolved board.
type BoardView struct {
	Present bool   `json:"present"`
	Value   string `json:"value,omitempty"`
	Source  string `json:"source"`
}

// PresenceView is the JSON-safe view of a credential: presence and source
// only — the value is never exposed.
type PresenceView struct {
	Present bool   `json:"present"`
	Source  string `json:"source"`
}

// ShowView is the resolved view rendered by `config show`.
type ShowView struct {
	Board BoardView    `json:"board"`
	Key   PresenceView `json:"key"`
	Token PresenceView `json:"token"`
}

// ShowView returns the resolved configuration as a serializable view that
// never contains credential values.
func (r Resolved) ShowView() ShowView {
	return ShowView{
		Board: BoardView{Present: r.Board.Present, Value: r.Board.Value, Source: r.Board.Source},
		Key:   PresenceView{Present: r.Key.Present, Source: r.Key.Source},
		Token: PresenceView{Present: r.Token.Present, Source: r.Token.Source},
	}
}
