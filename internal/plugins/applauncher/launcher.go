package applauncher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/adrg/xdg"
	"github.com/barab-i/incipio/pkgs/plugin"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-ini/ini"
	"go.uber.org/zap"
)

const Keyword = "!a"

var metadata = plugin.Metadata{
	Name:        "Application Launcher",
	Keyword:     Keyword,
	Flag:        "", // No specific flag needed as it's mandatory and default.
	IsMandatory: true,
	IsDefault:   true,
}

// DesktopEntry represents information parsed from a .desktop file.
type DesktopEntry struct {
	Name        string
	Exec        string
	Icon        string
	Comment     string
	FilePath    string
	GenericName string
	Keywords    string
}

// AppLauncherPlugin implements the plugin.Plugin interface for launching apps.
type AppLauncherPlugin struct {
	apps []DesktopEntry
}

// New creates a new instance of the AppLauncherPlugin.
func New() *AppLauncherPlugin {
	return &AppLauncherPlugin{}
}

// Metadata returns the plugin's metadata.
func (p *AppLauncherPlugin) Metadata() plugin.Metadata {
	return metadata
}

// Name returns the plugin's name.
func (p *AppLauncherPlugin) Name() string {
	return metadata.Name
}

// Keyword returns the plugin's keyword.
func (p *AppLauncherPlugin) Keyword() string {
	return metadata.Keyword
}

// Init scans for .desktop files during initialization.
func (p *AppLauncherPlugin) Init() tea.Cmd {
	p.scanDesktopFiles()
	return nil
}

const (
	scoreNamePrefix = 100
	scoreNameMatch  = 50
	scoreGeneric    = 30
	scoreKeyword    = 20
	scoreComment    = 10
	scoreExec       = 5
)

type scoredResult struct {
	Result plugin.Result
	Score  int
}

// GetResults filters and sorts applications based on query relevance.
func (p *AppLauncherPlugin) GetResults(query string) ([]plugin.Result, error) {
	lowerQuery := strings.ToLower(strings.TrimSpace(query))

	if lowerQuery == "" {
		results := make([]plugin.Result, len(p.apps))
		for i, app := range p.apps {
			results[i] = plugin.Result{
				Title:       app.Name,
				Description: app.Comment,
				Identifier:  app.FilePath,
			}
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].Title < results[j].Title
		})
		return results, nil
	}

	scoredResults := []scoredResult{}
	for _, app := range p.apps {
		score := calculateRelevanceScore(app, lowerQuery)
		if score > 0 {
			scoredResults = append(scoredResults, scoredResult{
				Result: plugin.Result{
					Title:       app.Name,
					Description: app.Comment,
					Identifier:  app.FilePath,
				},
				Score: score,
			})
		}
	}

	sort.SliceStable(scoredResults, func(i, j int) bool {
		if scoredResults[i].Score != scoredResults[j].Score {
			return scoredResults[i].Score > scoredResults[j].Score
		}
		return scoredResults[i].Result.Title < scoredResults[j].Result.Title
	})

	finalResults := make([]plugin.Result, len(scoredResults))
	for i, sr := range scoredResults {
		finalResults[i] = sr.Result
	}

	return finalResults, nil
}

func calculateRelevanceScore(app DesktopEntry, lowerQuery string) int {
	score := 0
	lowerName := strings.ToLower(app.Name)
	lowerGeneric := strings.ToLower(app.GenericName)
	lowerKeywords := strings.ToLower(app.Keywords)
	lowerComment := strings.ToLower(app.Comment)
	lowerExec := strings.ToLower(app.Exec)

	if strings.HasPrefix(lowerName, lowerQuery) {
		score = max(score, scoreNamePrefix)
	} else if strings.Contains(lowerName, lowerQuery) {
		score = max(score, scoreNameMatch)
	}

	if strings.Contains(lowerGeneric, lowerQuery) {
		score = max(score, scoreGeneric)
	}
	if strings.Contains(lowerKeywords, lowerQuery) {
		score = max(score, scoreKeyword)
	}
	if strings.Contains(lowerComment, lowerQuery) {
		score = max(score, scoreComment)
	}
	if strings.Contains(lowerExec, lowerQuery) {
		score = max(score, scoreExec)
	}

	// Ensure at least one field matched if score > 0
	if score > 0 && !(strings.Contains(lowerName, lowerQuery) ||
		strings.Contains(lowerGeneric, lowerQuery) ||
		strings.Contains(lowerKeywords, lowerQuery) ||
		strings.Contains(lowerComment, lowerQuery) ||
		strings.Contains(lowerExec, lowerQuery)) {
		return 0
	}

	return score
}

// Execute launches the application corresponding to the identifier (file path).
func (p *AppLauncherPlugin) Execute(identifier string) tea.Cmd {
	var targetApp *DesktopEntry
	for i := range p.apps {
		if p.apps[i].FilePath == identifier {
			targetApp = &p.apps[i]
			break
		}
	}

	if targetApp == nil {
		zap.L().Warn("Could not find app for execution.", zap.String("identifier", identifier))
		return nil
	}

	execParts := strings.Fields(targetApp.Exec)
	cleanedExec := []string{}
	for _, part := range execParts {
		if !strings.HasPrefix(part, "%") {
			cleanedExec = append(cleanedExec, part)
		}
	}
	if len(cleanedExec) == 0 {
		zap.L().Warn("Could not determine command from Exec field.",
			zap.String("execField", targetApp.Exec),
			zap.String("filePath", targetApp.FilePath))
		return nil
	}
	command := cleanedExec[0]
	args := cleanedExec[1:]

	cmd := exec.Command(command, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create a new session to detach from the terminal.
	}
	err := cmd.Start()

	if err != nil {
		zap.L().Error("Error executing command.",
			zap.String("command", targetApp.Exec),
			zap.String("filePath", targetApp.FilePath),
			zap.Error(err))
		return nil
	}

	return tea.Quit
}

// Update handles messages.
func (p *AppLauncherPlugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	return p, nil
}

// View returns an empty string as this plugin uses the main application's list view.
func (p *AppLauncherPlugin) View() string {
	return ""
}

// GetError returns nil as this plugin handles errors internally or via results.
func (p *AppLauncherPlugin) GetError() error {
	return nil
}

func (p *AppLauncherPlugin) scanDesktopFiles() {
	p.apps = []DesktopEntry{}
	appDirs := xdg.ApplicationDirs
	foundPaths := make(map[string]struct{})

	for _, dir := range appDirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					zap.L().Debug("Skipping inaccessible directory during desktop file scan.", zap.String("path", path), zap.Error(err))
					return filepath.SkipDir
				}
				zap.L().Debug("Skipping file due to error during desktop file scan.", zap.String("path", path), zap.Error(err))
				return nil
			}

			if !d.IsDir() && strings.HasSuffix(d.Name(), ".desktop") {
				absPath, absErr := filepath.Abs(path)
				if absErr != nil {
					zap.L().Debug("Could not get absolute path, using original.", zap.String("path", path), zap.Error(absErr))
					absPath = path
				}

				if _, found := foundPaths[absPath]; !found {
					entry, parseErr := parseDesktopFile(path)
					if parseErr == nil && entry != nil && shouldDisplayEntry(entry) {
						p.apps = append(p.apps, *entry)
						foundPaths[absPath] = struct{}{}
					} else if parseErr != nil {
						zap.L().Debug("Failed to parse .desktop file or entry not displayable.", zap.String("path", path), zap.Error(parseErr))
					}
				}
			}
			return nil
		})
		if err != nil {
			// Log the error from walking the directory but continue with other directories.
			zap.L().Warn("Error walking application directory for .desktop files.", zap.String("directory", dir), zap.Error(err))
		}
	}
}

func parseDesktopFile(filePath string) (*DesktopEntry, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{SkipUnrecognizableLines: true}, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load INI file '%s': %w", filePath, err)
	}

	section := cfg.Section("Desktop Entry")
	if !section.HasKey("Name") {
		return nil, fmt.Errorf("missing [Desktop Entry] section or Name key in '%s'", filePath)
	}

	entry := &DesktopEntry{
		Name:        section.Key("Name").String(),
		Exec:        section.Key("Exec").String(),
		Icon:        section.Key("Icon").String(),
		Comment:     section.Key("Comment").String(),
		GenericName: section.Key("GenericName").String(),
		Keywords:    section.Key("Keywords").String(),
		FilePath:    filePath,
	}

	if entry.Name == "" || entry.Exec == "" {
		return nil, fmt.Errorf("missing required field Name or Exec in '%s'", filePath)
	}

	return entry, nil
}

func shouldDisplayEntry(entry *DesktopEntry) bool {
	cfg, err := ini.LoadSources(ini.LoadOptions{SkipUnrecognizableLines: true}, entry.FilePath)
	if err != nil {
		zap.L().Debug("Could not reload .desktop file for display check, assuming displayable.", zap.String("path", entry.FilePath), zap.Error(err))
		return true // Default to displayable if re-parsing fails.
	}
	section := cfg.Section("Desktop Entry")

	if noDisplay, _ := section.Key("NoDisplay").Bool(); noDisplay {
		return false
	}
	if hidden, _ := section.Key("Hidden").Bool(); hidden {
		return false
	}

	return true
}
