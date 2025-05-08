package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/barab-i/incipio/pkgs/plugin"
	tea "github.com/charmbracelet/bubbletea"
)

// keyword is the activation keyword for this plugin.
const keyword = "!n"

// metadata defines the properties of the Nix Shell Runner plugin.
var metadata = plugin.Metadata{
	Name:    "Nix Shell Runner", // Name displayed in the application.
	Keyword: keyword,            // Keyword to activate this plugin.
	Flag:    "nixshell",         // Command-line flag to enable this optional plugin.
}

// NixShellPlugin implements the plugin.Plugin interface.
// It finds executables using `nix-locate` and allows running them via `nix shell`.
type NixShellPlugin struct {
	err           error           // Stores any error encountered during plugin operation.
	cachedResults []plugin.Result // Caches results from `nix-locate` for performance.
	resultsMutex  sync.RWMutex    // Protects access to cachedResults and err.
	isLoading     bool            // True if `nix-locate` is running and results are being loaded.
}

// New is the constructor for NixShellPlugin, called by the plugin loader (Yaegi).
// It must return a type that implements plugin.Plugin.
func New() plugin.Plugin {
	// Initialize the plugin with isLoading set to true,
	// as results will be fetched asynchronously.
	p := &NixShellPlugin{isLoading: true}
	return p
}

// Metadata returns the static metadata of the plugin.
func (p *NixShellPlugin) Metadata() plugin.Metadata {
	return metadata
}

// Name returns the display name of the plugin.
func (p *NixShellPlugin) Name() string {
	return metadata.Name
}

// Keyword returns the activation keyword for the plugin.
func (p *NixShellPlugin) Keyword() string {
	return metadata.Keyword
}

// loadNixLocateResults executes `nix-locate` to find available executables
// and populates the internal cache. This method is run in a goroutine.
func (p *NixShellPlugin) loadNixLocateResults() {
	// Ensure isLoading is set to false when this function exits,
	// regardless of success or failure.
	defer func() {
		p.resultsMutex.Lock()
		p.isLoading = false
		p.resultsMutex.Unlock()
	}()

	// Command to find executable files from top-level packages.
	cmd := exec.Command("nix-locate", "--type", "x", "--top-level", "bin/")
	var out bytes.Buffer
	var stderr bytes.Buffer // Capture stderr for more detailed error messages.
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	p.resultsMutex.Lock()
	defer p.resultsMutex.Unlock()

	if err != nil {
		// Format a detailed error message including nix-locate's stderr.
		errMsg := fmt.Sprintf("failed to run nix-locate: %v. Stderr: %s", err, stderr.String())
		p.err = fmt.Errorf("%s", errMsg)
		p.cachedResults = nil // Clear any potentially stale cache on error.
		return
	}

	// Process the output of nix-locate.
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// Pre-allocate slice for efficiency, assuming most lines are valid.
	results := make([]plugin.Result, 0, len(lines))

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			// Skip malformed lines.
			continue
		}

		// Example nix-locate output line:
		// nixpkgs.ripgrep.out /nix/store/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-ripgrep-13.0.0/bin/rg
		pkgAttrWithSuffix := parts[0]                            // e.g., "nixpkgs.ripgrep.out"
		pkgAttr := strings.TrimSuffix(pkgAttrWithSuffix, ".out") // e.g., "nixpkgs.ripgrep" or "ripgrep"
		fullPath := parts[len(parts)-1]                          // e.g., "/nix/store/.../bin/rg"
		executable := filepath.Base(fullPath)                    // e.g., "rg"

		var attrForShell string

		// Convert package attribute to the format required by `nix shell` (e.g., nixpkgs#ripgrep).
		attrForShell = "nixpkgs#" + pkgAttr

		// Command to be executed when the user selects this result.
		runCmd := fmt.Sprintf("nix shell %s -c %s", attrForShell, executable)
		// Description shown to the user, indicating the package attribute.

		results = append(results, plugin.Result{
			Identifier:  runCmd,     // The actual command string.
			Title:       executable, // The executable name.
			Description: pkgAttr,    // The package attribute.
		})
	}
	p.cachedResults = results
	p.err = nil // Clear any previous error on successful load.
}

// Init is called once when the plugin is loaded.
// It checks for the `nix-locate` dependency and starts the background loading of results.
func (p *NixShellPlugin) Init() tea.Cmd {
	// Check if `nix-locate` is available in the system's PATH.
	_, err := exec.LookPath("nix-locate")
	if err != nil {
		p.resultsMutex.Lock()
		p.err = fmt.Errorf("'nix-locate' command not found in PATH. Nix plugin disabled: %w", err)
		p.isLoading = false // Mark loading as finished due to this error.
		p.resultsMutex.Unlock()
		return func() tea.Msg { return nil } // Return a no-op command.
	}

	// Start loading nix-locate results in a separate goroutine
	// to avoid blocking the main application startup.
	go p.loadNixLocateResults()

	return func() tea.Msg { return nil } // Return a no-op command.
}

// GetResults is called by the application to fetch results based on the user's query.
// It filters the cached `nix-locate` results.
func (p *NixShellPlugin) GetResults(query string) ([]plugin.Result, error) {
	p.resultsMutex.RLock() // Use RLock for reading shared state (err, isLoading).

	// If an error occurred during initialization or loading, display it as a result.
	if p.err != nil {
		defer p.resultsMutex.RUnlock()
		return []plugin.Result{
			{Title: "Nix Plugin Error", Description: p.err.Error(), Identifier: "nix_internal_error"},
		}, nil
	}

	// If results are still being loaded, show a loading message.
	if p.isLoading {
		defer p.resultsMutex.RUnlock()
		return []plugin.Result{
			{Title: "Loading Nix packages via nix-locate...", Description: "Please wait...", Identifier: "nix_loading"},
		}, nil
	}
	p.resultsMutex.RUnlock() // Unlock early if no error and not loading.

	p.resultsMutex.RLock() // Re-lock for reading cachedResults.
	defer p.resultsMutex.RUnlock()

	searchQuery := strings.ToLower(strings.TrimSpace(query))

	// If the search query is empty, provide an informational message.
	if searchQuery == "" {
		return []plugin.Result{
			{Title: "Nix Shell Runner", Description: "Enter command to search via nix-locate.", Identifier: "nix_info_ready"},
		}, nil
	}

	// If the cache is unexpectedly nil (should not happen if isLoading is false and no error),
	// return an appropriate message.
	if p.cachedResults == nil {
		return []plugin.Result{
			{Title: "Nix results not available", Description: "Cache is empty.", Identifier: "nix_cache_empty"},
		}, nil
	}

	filteredResults := make([]plugin.Result, 0)
	for _, result := range p.cachedResults {
		// Match against the executable name (Title) or package attribute (Description).
		if strings.Contains(strings.ToLower(result.Title), searchQuery) ||
			strings.Contains(strings.ToLower(result.Description), searchQuery) {
			filteredResults = append(filteredResults, result)
		}
	}

	// If no results match the query.
	if len(filteredResults) == 0 {
		return []plugin.Result{
			{Title: "No results found", Description: fmt.Sprintf("For query: '%s'", query), Identifier: "nix_no_results"},
		}, nil
	}

	return filteredResults, nil
}

// Execute is called when the user selects a result.
// The `identifier` is the command string generated in `loadNixLocateResults`.
func (p *NixShellPlugin) Execute(identifier string) tea.Cmd {
	// Define placeholder identifiers that should not be executed.
	placeholders := map[string]struct{}{
		"nix_loading":        {},
		"nix_internal_error": {}, // Matches the error identifier from GetResults.
		"nix_info_ready":     {},
		"nix_no_results":     {},
		"nix_cache_empty":    {},
	}
	if _, isPlaceholder := placeholders[identifier]; isPlaceholder {
		return func() tea.Msg { return nil } // Do nothing for placeholder items.
	}

	parts := strings.Fields(identifier)
	// Validate the command structure (e.g., "nix shell nixpkgs#ripgrep -c rg").
	// Expects at least 5 parts: "nix", "shell", "<package_attr>", "-c", "<executable>"
	if len(parts) < 5 || parts[0] != "nix" || parts[1] != "shell" || parts[len(parts)-2] != "-c" {
		p.resultsMutex.Lock()
		p.err = fmt.Errorf("invalid command identifier format for execution: %s", identifier)
		p.resultsMutex.Unlock()
		// The error will be displayed by GetResults on the next update.
		return func() tea.Msg { return nil }
	}

	// Prepare the command for execution.
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = nil  // Detach from standard input.
	cmd.Stdout = nil // Detach from standard output.
	cmd.Stderr = nil // Detach from standard error.
	// Create a new session to detach the process from the Incipio terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	err := cmd.Start() // Start the command asynchronously.
	if err != nil {
		p.resultsMutex.Lock()
		p.err = fmt.Errorf("failed to start command '%s': %w", identifier, err)
		p.resultsMutex.Unlock()
		// The error will be displayed by GetResults on the next update.
		return func() tea.Msg { return nil }
	}

	// If the command starts successfully, quit the Incipio application.
	return tea.Quit
}

// Update handles messages from the Bubble Tea runtime.
// This plugin does not currently process any specific messages.
func (p *NixShellPlugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	return p, func() tea.Msg { return nil } // Return self and a no-op command.
}

// View is responsible for rendering the plugin's UI.
// This plugin uses the main application's list view, so it returns an empty string.
func (p *NixShellPlugin) View() string {
	return ""
}

// GetError returns any error encountered by the plugin.
// This can be used by the main application to display error information.
func (p *NixShellPlugin) GetError() error {
	p.resultsMutex.RLock()
	defer p.resultsMutex.RUnlock()
	return p.err
}
