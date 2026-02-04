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

// Start launches the AI Core process (gemini or aider)
func (m *Manager) Start(kernel string, port int) error {
	var cmd *exec.Cmd
	var serverPath string

	if kernel == "aider" {
		serverPath = filepath.Join(m.WorkDir, "cores", "aider")
		log.Info().Str("kernel", "aider").Str("path", serverPath).Int("port", port).Msg("Starting Aider Core...")

		// Check if server.py exists
		if _, err := os.Stat(filepath.Join(serverPath, "server.py")); os.IsNotExist(err) {
			return fmt.Errorf("aider server script not found at %s", serverPath)
		}

		// Run python server.py using venv
		// We use the venv created in the cores/aider directory
		if runtime.GOOS == "windows" {
			pythonPath := filepath.Join(serverPath, ".venv", "Scripts", "python.exe")
			// Fallback to system python if venv not found (though it should be there)
			if _, err := os.Stat(pythonPath); err == nil {
				cmd = exec.Command(pythonPath, "server.py")
			} else {
				log.Warn().Msg("Aider venv not found, falling back to system python")
				cmd = exec.Command("python", "server.py")
			}
		} else {
			pythonPath := filepath.Join(serverPath, ".venv", "bin", "python3")
			if _, err := os.Stat(pythonPath); err == nil {
				cmd = exec.Command(pythonPath, "server.py")
			} else {
				log.Warn().Msg("Aider venv not found, falling back to system python3")
				cmd = exec.Command("python3", "server.py")
			}
		}
		cmd.Dir = serverPath

		// Inject Environment Variables
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", port))
		// PYTHONPATH might be needed if not set
		cmd.Env = append(cmd.Env, "PYTHONPATH=.")

	} else {
		// Default to Gemini
		serverPath = filepath.Join(m.WorkDir, "cores", "gemini", "packages", "a2a-server")

		// Check if directory exists
		if _, err := os.Stat(serverPath); os.IsNotExist(err) {
			return fmt.Errorf("gemini core path not found: %s", serverPath)
		}

		log.Info().Str("kernel", "gemini").Str("path", serverPath).Int("port", port).Msg("Starting Gemini Core...")

		if runtime.GOOS == "windows" {
			cmd = exec.Command("npm.cmd", "run", "start")
		} else {
			cmd = exec.Command("npm", "run", "start")
		}
		cmd.Dir = serverPath

		// Inject Environment Variables
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", port))
		// Pass CODER_AGENT_PORT for Gemini specifically as it uses it
		cmd.Env = append(cmd.Env, fmt.Sprintf("CODER_AGENT_PORT=%d", port))
	}

	// Shared startup logic
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s process: %w", kernel, err)
	}

	m.cmd = cmd

	// Async Log Forwarding
	go scanLog(stdout, fmt.Sprintf("%s_OUT", kernel))
	go scanLog(stderr, fmt.Sprintf("%s_ERR", kernel))

	log.Info().Str("kernel", kernel).Int("pid", cmd.Process.Pid).Msg("Core Started")
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
