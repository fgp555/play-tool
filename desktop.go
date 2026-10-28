package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend
var desktopFiles embed.FS

type desktopHandler struct {
	app         *application.App
	initialPath string
	assets      http.Handler
}

type desktopProject struct {
	Path          string `json:"path"`
	GradleRoot    string `json:"gradleRoot"`
	Package       string `json:"package"`
	Track         string `json:"track"`
	Config        string `json:"config"`
	KeyConfigured bool   `json:"keyConfigured"`
	Configuration string `json:"configuration"`
	Error         string `json:"error,omitempty"`
}

type desktopRunRequest struct {
	Action  string `json:"action"`
	Path    string `json:"path"`
	KeyPath string `json:"keyPath"`
}

func runDesktopUI() error {
	startPath, err := os.Getwd()
	if err != nil {
		return err
	}
	assetsFS, err := fs.Sub(desktopFiles, "frontend")
	if err != nil {
		return fmt.Errorf("no se pudieron cargar los recursos de la interfaz: %w", err)
	}
	handler := &desktopHandler{
		initialPath: startPath,
		assets:      http.FileServer(http.FS(assetsFS)),
	}
	app := application.New(application.Options{
		Name:        "play-tool",
		Description: "Herramientas locales para proyectos Android y Google Play",
		Assets: application.AssetOptions{
			Handler: handler,
		},
	})
	handler.app = app
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "play-tool",
		Width:  1040,
		Height: 760,
		URL:    "/",
	})
	return app.Run()
}

func (h *desktopHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/project":
		h.handleProject(w, r)
	case "/api/pick-folder":
		h.handlePickFolder(w, r)
	case "/api/pick-key":
		h.handlePickKey(w, r)
	case "/api/run":
		h.handleRun(w, r)
	default:
		h.assets.ServeHTTP(w, r)
	}
}

func (h *desktopHandler) handleProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	projectPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if projectPath == "" {
		projectPath = h.initialPath
	}
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		writeDesktopJSON(w, http.StatusBadRequest, desktopProject{Error: err.Error()})
		return
	}
	cfg, err := loadConfigFrom(projectPath)
	if err != nil {
		writeDesktopJSON(w, http.StatusOK, desktopProject{Path: projectPath, Error: err.Error()})
		return
	}
	config := cfg.ConfigPath
	if config == "" {
		config = "automática (sin play-tool.json)"
	}
	keyConfigured := false
	if cfg.Key != "" {
		if info, statErr := os.Stat(cfg.Key); statErr == nil && info.Mode().IsRegular() {
			keyConfigured = true
		}
	}
	configuration := "automática"
	if cfg.ConfigPath != "" {
		configuration = "play-tool.json"
	}
	writeDesktopJSON(w, http.StatusOK, desktopProject{
		Path:          projectPath,
		GradleRoot:    cfg.GradleRoot,
		Package:       cfg.Pkg,
		Track:         cfg.Track,
		Config:        config,
		KeyConfigured: keyConfigured,
		Configuration: configuration,
	})
}

func (h *desktopHandler) handlePickFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	path, err := h.app.Dialog.OpenFile().
		SetTitle("Selecciona la carpeta del proyecto Android").
		SetDirectory(h.initialPath).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
	if err != nil {
		writeDesktopJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeDesktopJSON(w, http.StatusOK, map[string]string{"path": path})
}

func (h *desktopHandler) handlePickKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	path, err := h.app.Dialog.OpenFile().
		SetTitle("Selecciona la clave de servicio de Google Play").
		AddFilter("Archivos JSON", "*.json").
		CanChooseDirectories(false).
		CanChooseFiles(true).
		PromptForSingleSelection()
	if err != nil {
		writeDesktopJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeDesktopJSON(w, http.StatusOK, map[string]string{"path": path})
}

func (h *desktopHandler) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var request desktopRunRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "solicitud inválida", http.StatusBadRequest)
		return
	}
	args, ok := desktopActionArgs(request.Action)
	if !ok {
		http.Error(w, "acción no válida", http.StatusBadRequest)
		return
	}
	projectPath := strings.TrimSpace(request.Path)
	if projectPath == "" {
		projectPath = h.initialPath
	}
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg, err := loadConfigFrom(projectPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	executable, err := os.Executable()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cmd := exec.Command(executable, args...)
	cmd.Dir = projectPath
	if keyPath := strings.TrimSpace(request.KeyPath); keyPath != "" {
		cmd.Env = setEnvironment(os.Environ(), "PLAY_TOOL_KEY", resolve(cfg.GradleRoot, keyPath))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	flusher, canFlush := w.(http.Flusher)
	output := make(chan string)
	var readers sync.WaitGroup
	readPipe := func(reader io.Reader) {
		defer readers.Done()
		buffered := bufio.NewReader(reader)
		for {
			line, readErr := buffered.ReadString('\n')
			if line != "" {
				output <- line
			}
			if readErr != nil {
				return
			}
		}
	}
	readers.Add(2)
	go readPipe(stdout)
	go readPipe(stderr)
	go func() {
		readers.Wait()
		close(output)
	}()
	for line := range output {
		_, _ = io.WriteString(w, line)
		if canFlush {
			flusher.Flush()
		}
	}
	if err := cmd.Wait(); err != nil {
		_, _ = fmt.Fprintf(w, "\n❌ Error: %v\n", err)
		return
	}
	_, _ = io.WriteString(w, "\n✅ Acción completada.\n")
}

func desktopActionArgs(action string) ([]string, bool) {
	switch action {
	case "show", "build", "upload", "deploy", "assembleDebug", "assembleRelease", "bundleRelease", "installDebug", "installRelease":
		return []string{action}, true
	case "infoDebug":
		return []string{"info", "assembleDebug"}, true
	case "infoRelease":
		return []string{"info", "assembleRelease"}, true
	case "infoBundle":
		return []string{"info", "bundleRelease"}, true
	default:
		return nil, false
	}
}

func setEnvironment(environment []string, name, value string) []string {
	prefix := strings.ToUpper(name) + "="
	result := make([]string, 0, len(environment)+1)
	for _, variable := range environment {
		if !strings.HasPrefix(strings.ToUpper(variable), prefix) {
			result = append(result, variable)
		}
	}
	return append(result, name+"="+value)
}

func writeDesktopJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
