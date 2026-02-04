package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog/log"
)

// Manager handles the lifecycle of the Gemini Core process
type Manager struct {
	cmd     *exec.Cmd
	WorkDir string
}

func NewManager(workDir string) *Manager {
	return &Manager{
		WorkDir: workDir,
	}
}

// Start launches the Gemini Core process
// It tries to find the executable or falls back to 'npm start' for dev
func (m *Manager) Start(port int) error {
	serverPath := filepath.Join(m.WorkDir, "cores", "gemini", "packages", "a2a-server")

	// Check if directory exists
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		return fmt.Errorf("gemini core path not found: %s", serverPath)
	}

	log.Info().Str("path", serverPath).Int("port", port).Msg("Starting Gemini Core...")

	// Construct command (Dev Mode: npm start)
	// In production this should run the compiled binary
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("npm.cmd", "run", "start")
	} else {
		cmd = exec.Command("npm", "run", "start")
	}

	cmd.Dir = serverPath

	// Inject Environment Variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", port))
	// Critical: Force IPv4 binding if needed, though '0.0.0.0' in app.ts handles it
	// cmd.Env = append(cmd.Env, "HOST=0.0.0.0")

	// Pipe pipes
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gemini process: %w", err)
	}

	m.cmd = cmd

	// Async Log Forwarding
	go scanLog(stdout, "GEMINI_OUT")
	go scanLog(stderr, "GEMINI_ERR")

	log.Info().Int("pid", cmd.Process.Pid).Msg("Gemini Core Started")
	return nil
}

// Stop terminates the process
func (m *Manager) Stop() error {
	if m.cmd != nil && m.cmd.Process != nil {
		log.Info().Msg("Stopping Gemini Core...")
		if runtime.GOOS == "windows" {
			// /F = Force, /T = Tree (kill child processes)
			err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(m.cmd.Process.Pid)).Run()
			if err != nil {
				return fmt.Errorf("failed to kill process on windows: %w", err)
			}
		} else {
			if err := m.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	}
	return nil
}

func scanLog(r io.Reader, prefix string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Info().Str("stream", prefix).Msg(scanner.Text())
	}
}
