package backend

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aleksclark/spec-torture/arp/a2a"
	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// ExecBackend launches agents as local OS processes using the template command.
// The port is injected via the template's port_env variable and ARP_TOKEN
// carries the child token. Readiness is determined by polling the health path.
type ExecBackend struct {
	// Shell used to run the template command. Defaults to "/bin/sh -c".
	Shell []string
	a2a   *a2a.Client
}

// NewExecBackend returns an ExecBackend with default settings.
func NewExecBackend() *ExecBackend {
	return &ExecBackend{
		Shell: []string{"/bin/sh", "-c"},
		a2a:   a2a.NewClient(5 * time.Second),
	}
}

type execHandle struct {
	cmd       *exec.Cmd
	directURL string
	port      int
}

func (h *execHandle) DirectURL() string { return h.directURL }
func (h *execHandle) Port() int         { return h.port }
func (h *execHandle) PID() int {
	if h.cmd == nil || h.cmd.Process == nil {
		return 0
	}
	return h.cmd.Process.Pid
}

func (h *execHandle) Stop(ctx context.Context, graceMs int) error {
	if h.cmd == nil || h.cmd.Process == nil {
		return nil
	}
	if graceMs <= 0 {
		graceMs = 5000
	}
	_ = h.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- h.cmd.Wait() }()
	select {
	case <-done:
		return nil
	case <-time.After(time.Duration(graceMs) * time.Millisecond):
		_ = h.cmd.Process.Kill()
		<-done
		return nil
	case <-ctx.Done():
		_ = h.cmd.Process.Kill()
		<-done
		return ctx.Err()
	}
}

// Spawn starts the process and blocks until the agent passes its health check.
func (b *ExecBackend) Spawn(ctx context.Context, spec SpawnSpec) (Handle, error) {
	if spec.Template.GetCommand() == "" {
		return nil, fmt.Errorf("template %q has no command", spec.Template.GetName())
	}
	shell := b.Shell
	if len(shell) == 0 {
		shell = []string{"/bin/sh", "-c"}
	}
	args := append(append([]string{}, shell[1:]...), spec.Template.GetCommand())
	cmd := exec.Command(shell[0], args...)
	cmd.Dir = spec.WorkspaceDir
	cmd.Env = b.buildEnv(spec)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start agent process: %w", err)
	}
	directURL := fmt.Sprintf("http://127.0.0.1:%d", spec.Port)
	h := &execHandle{cmd: cmd, directURL: directURL, port: spec.Port}

	path, interval, timeout, retries := HealthDefaults(spec.Template)
	hc := a2a.NewClient(timeout)
	for attempt := 0; attempt < retries; attempt++ {
		select {
		case <-ctx.Done():
			_ = h.Stop(context.Background(), 1000)
			return nil, ctx.Err()
		default:
		}
		if hc.HealthCheck(ctx, directURL, path) {
			return h, nil
		}
		time.Sleep(interval)
	}
	_ = h.Stop(context.Background(), 1000)
	return nil, fmt.Errorf("agent %q failed health check after %d attempts", spec.AgentID, retries)
}

func (b *ExecBackend) buildEnv(spec SpawnSpec) []string {
	env := os.Environ()
	set := func(k, v string) { env = append(env, k+"="+v) }
	portEnv := spec.Template.GetPortEnv()
	if portEnv == "" {
		portEnv = "A2A_PORT"
	}
	set(portEnv, strconv.Itoa(spec.Port))
	set("PORT", strconv.Itoa(spec.Port))
	if spec.Token != "" {
		set("ARP_TOKEN", spec.Token)
	}
	for k, v := range spec.Template.GetEnv() {
		set(k, v)
	}
	for k, v := range spec.Env {
		set(k, v)
	}
	return env
}

// FreePort asks the OS for a free TCP port. Used by callers that need a port
// before constructing a SpawnSpec.
func FreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// SkillTags returns the capability/skill tags advertised by a template, used
// for discovery matching.
func SkillTags(t *arpv1.AgentTemplate) []string {
	var tags []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			tags = append(tags, s)
		}
	}
	for _, c := range t.GetCapabilities() {
		add(c)
	}
	if cfg := t.GetA2ACardConfig(); cfg != nil {
		for _, sk := range cfg.GetSkills() {
			for _, tg := range sk.GetTags() {
				add(tg)
			}
		}
	}
	return tags
}
