package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestWalkUpPerKeyNearestWins(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	sub := filepath.Join(proj, "sub")
	writeConfig(t, filepath.Join(home, ".trello.yaml"), "board: home-board\nkey: home-key\ntoken: home-token\n")
	writeConfig(t, filepath.Join(proj, ".trello.yaml"), "board: proj-board\nkey: proj-key\n")
	writeConfig(t, filepath.Join(sub, ".trello.yml"), "board: sub-board\n")

	r := NewResolver(sub, home, filepath.Join(home, ".config", "trello"), nil)
	res, err := r.Resolve(Credentials{}, Env{})
	if err != nil {
		t.Fatal(err)
	}

	// board: nearest file (sub) wins
	if res.Board.Value != "sub-board" || res.Board.Source != filepath.Join(sub, ".trello.yml") {
		t.Errorf("board = %+v, want sub-board from %s", res.Board, filepath.Join(sub, ".trello.yml"))
	}
	// key: sub has no key, so proj wins
	if res.Key.Value != "proj-key" || res.Key.Source != filepath.Join(proj, ".trello.yaml") {
		t.Errorf("key = %+v, want proj-key from %s", res.Key, filepath.Join(proj, ".trello.yaml"))
	}
	// token: only home defines it
	if res.Token.Value != "home-token" || res.Token.Source != filepath.Join(home, ".trello.yaml") {
		t.Errorf("token = %+v, want home-token from %s", res.Token, filepath.Join(home, ".trello.yaml"))
	}
}

func TestPrecedenceFlagsEnvConfig(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	writeConfig(t, filepath.Join(proj, ".trello.yaml"), "board: cfg-board\nkey: cfg-key\ntoken: cfg-token\n")
	r := NewResolver(proj, home, filepath.Join(home, ".config", "trello"), nil)

	// Flags beat env and config.
	res, err := r.Resolve(Credentials{Board: "flag-board", Key: "flag-key", Token: "flag-token"}, Env{APIKey: "env-key", Token: "env-token"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Key.Value != "flag-key" || res.Key.Source != "flag" {
		t.Errorf("key = %+v, want flag", res.Key)
	}
	if res.Token.Value != "flag-token" || res.Token.Source != "flag" {
		t.Errorf("token = %+v, want flag", res.Token)
	}
	if res.Board.Value != "flag-board" || res.Board.Source != "flag" {
		t.Errorf("board = %+v, want flag", res.Board)
	}

	// Env beats config.
	res, err = r.Resolve(Credentials{}, Env{APIKey: "env-key", Token: "env-token"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Key.Value != "env-key" || res.Key.Source != "environment" {
		t.Errorf("key = %+v, want environment", res.Key)
	}
	if res.Token.Value != "env-token" || res.Token.Source != "environment" {
		t.Errorf("token = %+v, want environment", res.Token)
	}

	// Config is used when no flag/env.
	res, err = r.Resolve(Credentials{}, Env{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Key.Value != "cfg-key" || res.Key.Source != filepath.Join(proj, ".trello.yaml") {
		t.Errorf("key = %+v, want config file", res.Key)
	}

	// Board never comes from the environment.
	res, err = r.Resolve(Credentials{}, Env{APIKey: "x", Token: "y"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Board.Value != "cfg-board" {
		t.Errorf("board = %+v, want cfg-board from config", res.Board)
	}
}

func TestGlobalConfigUsedWhenNoProject(t *testing.T) {
	home := t.TempDir()
	global := filepath.Join(home, ".config", "trello")
	writeConfig(t, filepath.Join(global, "config.yaml"), "board: global-board\nkey: global-key\ntoken: global-token\n")
	r := NewResolver(home, home, global, nil)

	res, err := r.Resolve(Credentials{}, Env{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Board.Value != "global-board" || res.Board.Source != filepath.Join(global, "config.yaml") {
		t.Errorf("board = %+v, want global config", res.Board)
	}
	if res.Key.Value != "global-key" || res.Token.Value != "global-token" {
		t.Errorf("creds = %+v / %+v, want global", res.Key, res.Token)
	}
}

func TestProjectBeatsGlobal(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	global := filepath.Join(home, ".config", "trello")
	writeConfig(t, filepath.Join(proj, ".trello.yaml"), "board: proj-board\n")
	writeConfig(t, filepath.Join(global, "config.yaml"), "board: global-board\nkey: global-key\n")
	r := NewResolver(proj, home, global, nil)

	res, err := r.Resolve(Credentials{}, Env{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Board.Value != "proj-board" {
		t.Errorf("board = %+v, want proj-board", res.Board)
	}
	// key falls through to global.
	if res.Key.Value != "global-key" || res.Key.Source != filepath.Join(global, "config.yaml") {
		t.Errorf("key = %+v, want global-key", res.Key)
	}
}

func TestUnknownKeyWarns(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, filepath.Join(home, ".trello.yaml"), "board: b\nbord: typo\nkey: k\n")

	var warnings []string
	r := NewResolver(home, home, "", func(msg string) { warnings = append(warnings, msg) })
	if _, err := r.Resolve(Credentials{}, Env{}); err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "bord") {
		t.Errorf("warning should mention the unknown key, got %q", warnings[0])
	}
}

func TestUnknownKeyDoesNotFail(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, filepath.Join(home, ".trello.yaml"), "board: b\nbogus: 1\n")
	r := NewResolver(home, home, "", nil)
	res, err := r.Resolve(Credentials{}, Env{})
	if err != nil {
		t.Fatalf("unknown keys must not fail resolution: %v", err)
	}
	if res.Board.Value != "b" {
		t.Errorf("board = %+v, want b", res.Board)
	}
}

func TestInvalidYAMLErrors(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, filepath.Join(home, ".trello.yaml"), "board: [unclosed\n")
	r := NewResolver(home, home, "", nil)
	if _, err := r.Resolve(Credentials{}, Env{}); err == nil {
		t.Fatal("expected an error for invalid YAML")
	}
}

func TestNonMappingConfigErrors(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, filepath.Join(home, ".trello.yaml"), "- just\n- a list\n")
	r := NewResolver(home, home, "", nil)
	if _, err := r.Resolve(Credentials{}, Env{}); err == nil {
		t.Fatal("expected an error for a non-mapping config")
	}
}

func TestPaths(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	writeConfig(t, filepath.Join(proj, ".trello.yaml"), "board: b\n")
	r := NewResolver(filepath.Join(proj, "sub"), home, filepath.Join(home, ".config", "trello"), nil)

	project, global := r.Paths()
	if project != filepath.Join(proj, ".trello.yaml") {
		t.Errorf("project = %q, want %q", project, filepath.Join(proj, ".trello.yaml"))
	}
	if global != filepath.Join(home, ".config", "trello", "config.yaml") {
		t.Errorf("global = %q, want %q", global, filepath.Join(home, ".config", "trello", "config.yaml"))
	}
}

func TestPathsWouldBeInCWD(t *testing.T) {
	home := t.TempDir()
	r := NewResolver(home, home, filepath.Join(home, ".config", "trello"), nil)
	project, _ := r.Paths()
	if project != filepath.Join(home, ".trello.yaml") {
		t.Errorf("project = %q, want %q", project, filepath.Join(home, ".trello.yaml"))
	}
}

func TestInitCreatesFile(t *testing.T) {
	dir := t.TempDir()
	r := NewResolver(dir, dir, "", nil)
	path, err := r.Init(InitInput{Board: "my-board", Key: "k", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, ".trello.yaml") {
		t.Errorf("path = %q", path)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("permissions = %o, want 600", fi.Mode().Perm())
	}

	var m map[string]any
	if err := yaml.Unmarshal([]byte(readFile(t, path)), &m); err != nil {
		t.Fatal(err)
	}
	if m["board"] != "my-board" || m["key"] != "k" || m["token"] != "t" {
		t.Errorf("unexpected content: %v", m)
	}
}

func TestInitMergesExistingFile(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, filepath.Join(dir, ".trello.yaml"), "board: old\ncustom: keep-me\n")
	r := NewResolver(dir, dir, "", nil)
	if _, err := r.Init(InitInput{Board: "new-board"}); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := yaml.Unmarshal([]byte(readFile(t, filepath.Join(dir, ".trello.yaml"))), &m); err != nil {
		t.Fatal(err)
	}
	if m["board"] != "new-board" {
		t.Errorf("board = %v, want new-board", m["board"])
	}
	if m["custom"] != "keep-me" {
		t.Errorf("custom = %v, want keep-me (preserved)", m["custom"])
	}
}

func TestInitRequiresBoard(t *testing.T) {
	r := NewResolver(t.TempDir(), "", "", nil)
	if _, err := r.Init(InitInput{}); err == nil {
		t.Fatal("expected an error when no board is given")
	}
}

func TestShowViewNeverExposesToken(t *testing.T) {
	res := Resolved{
		Board: Value{Present: true, Value: "my-board", Source: "/x/.trello.yaml"},
		Key:   Value{Present: true, Value: "secret-key", Source: "flag"},
		Token: Value{Present: true, Value: "secret-token", Source: "environment"},
	}
	view := res.ShowView()
	if view.Board.Value != "my-board" {
		t.Errorf("board value = %q", view.Board.Value)
	}
	if view.Key.Present != true || view.Key.Source != "flag" {
		t.Errorf("key view = %+v", view.Key)
	}
	if view.Token.Present != true || view.Token.Source != "environment" {
		t.Errorf("token view = %+v", view.Token)
	}
	// The ShowView struct has no field that can carry the token value.
	raw, err := yaml.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-token") || strings.Contains(string(raw), "secret-key") {
		t.Errorf("ShowView leaked credential values: %s", raw)
	}
}
