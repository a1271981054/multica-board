package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultBoardRepo = "a1271981054/multica-board"

type BoardVersionResponse struct {
	Current         string `json:"current"`
	Latest          string `json:"latest,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
	Message         string `json:"message,omitempty"`
}

type BoardUpdateStatusResponse struct {
	Status    string `json:"status"`
	Current   string `json:"current,omitempty"`
	Latest    string `json:"latest,omitempty"`
	Message   string `json:"message,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func boardInstallDir() string {
	if dir := strings.TrimSpace(os.Getenv("MULTICA_BOARD_INSTALL_DIR")); dir != "" {
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		// <install>/bin/server -> <install>
		if dir := filepath.Dir(filepath.Dir(exe)); dir != "" {
			return dir
		}
	}
	return ""
}

func boardVersionFile() string {
	dir := boardInstallDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "VERSION")
}

func readBoardVersion() string {
	file := boardVersionFile()
	if file == "" {
		return "dev"
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(raw))
}

func fetchLatestBoardRelease(repo string) (version, releaseURL string, err error) {
	if repo == "" {
		repo = defaultBoardRepo
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, reqErr := http.NewRequest(http.MethodGet, url, nil)
	if reqErr != nil {
		return "", "", reqErr
	}
	req.Header.Set("User-Agent", "Multica-Board")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, doErr := client.Do(req)
	if doErr != nil {
		return "", "", doErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github returned %s", resp.Status)
	}
	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", err
	}
	tag := strings.TrimPrefix(payload.TagName, "multica-board-v")
	if tag == payload.TagName {
		tag = strings.TrimPrefix(payload.TagName, "v")
	}
	return tag, payload.HTMLURL, nil
}

func compareBoardVersions(a, b string) int {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv int
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// GetBoardVersion reports the installed Board version and whether a newer
// GitHub release exists. It is a local convenience for the Settings page and
// the startup notice; failures degrade to "no update" so a network hiccup
// never blocks the board.
func (h *Handler) GetBoardVersion(w http.ResponseWriter, r *http.Request) {
	current := readBoardVersion()
	repo := strings.TrimSpace(os.Getenv("MULTICA_BOARD_REPO"))
	latest, releaseURL, err := fetchLatestBoardRelease(repo)
	if err != nil {
		writeJSON(w, http.StatusOK, BoardVersionResponse{
			Current: current,
			Message: "暂时无法检查更新",
		})
		return
	}
	writeJSON(w, http.StatusOK, BoardVersionResponse{
		Current:         current,
		Latest:          latest,
		UpdateAvailable: compareBoardVersions(current, latest) < 0,
		ReleaseURL:      releaseURL,
	})
}

// StartBoardUpdate downloads and applies the latest release in the
// background. The updater runs detached because it stops the web/backend
// services while replacing the bundle.
func (h *Handler) StartBoardUpdate(w http.ResponseWriter, r *http.Request) {
	installDir := boardInstallDir()
	cli := filepath.Join(installDir, "multica-board")
	if _, err := os.Stat(cli); err != nil {
		writeError(w, http.StatusBadRequest, "multica-board CLI not found")
		return
	}
	home := strings.TrimSpace(os.Getenv("MULTICA_BOARD_HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "cannot resolve board home")
			return
		}
		home = filepath.Join(userHome, "Library", "Application Support", "Multica Board")
	}
	updateDir := filepath.Join(home, "updates")
	if err := os.MkdirAll(updateDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "cannot create update directory")
		return
	}
	logFile, err := os.OpenFile(filepath.Join(updateDir, "update.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot open update log")
		return
	}
	cmd := exec.Command(cli, "update", "--apply")
	cmd.Env = os.Environ()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		writeError(w, http.StatusInternalServerError, "failed to start updater")
		return
	}
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
	}()
	writeJSON(w, http.StatusAccepted, map[string]any{"started": true})
}

// GetBoardUpdateStatus returns the updater's progress file. Missing file means
// no update has been started in this session.
func (h *Handler) GetBoardUpdateStatus(w http.ResponseWriter, r *http.Request) {
	home := strings.TrimSpace(os.Getenv("MULTICA_BOARD_HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			writeJSON(w, http.StatusOK, BoardUpdateStatusResponse{Status: "idle"})
			return
		}
		home = filepath.Join(userHome, "Library", "Application Support", "Multica Board")
	}
	raw, err := os.ReadFile(filepath.Join(home, "updates", "status.json"))
	if err != nil {
		writeJSON(w, http.StatusOK, BoardUpdateStatusResponse{Status: "idle", Current: readBoardVersion()})
		return
	}
	var status BoardUpdateStatusResponse
	if json.Unmarshal(raw, &status) != nil {
		writeJSON(w, http.StatusOK, BoardUpdateStatusResponse{Status: "idle", Current: readBoardVersion()})
		return
	}
	if status.Current == "" {
		status.Current = readBoardVersion()
	}
	writeJSON(w, http.StatusOK, status)
}
