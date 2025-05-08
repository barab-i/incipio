package pluginmanager

import (
	"fmt"
	"sort"
	"strings"

	"github.com/barab-i/incipio/internal/app"
	"github.com/barab-i/incipio/pkgs/plugin"
	tea "github.com/charmbracelet/bubbletea"
)

const keyword = "!p"

var metadata = plugin.Metadata{
	Name:        "Plugin Manager",
	Keyword:     keyword,
	Flag:        "", // Mandatory plugins don't need a command-line flag.
	IsMandatory: true,
	IsDefault:   false,
}

// PluginManagerPlugin displays the status of registered plugins.
type PluginManagerPlugin struct {
	mainPluginManager *app.PluginManager // Reference to the main application's plugin manager.
}

// New creates a new instance of the PluginManagerPlugin.
// It requires the main PluginManager to access the list of loaded plugins.
func New(mainPM *app.PluginManager) *PluginManagerPlugin {
	if mainPM == nil {
		panic("PluginManagerPlugin requires a non-nil main PluginManager")
	}
	return &PluginManagerPlugin{
		mainPluginManager: mainPM,
	}
}

// Metadata returns the plugin's metadata.
func (p *PluginManagerPlugin) Metadata() plugin.Metadata {
	return metadata
}

// Name returns the plugin's name.
func (p *PluginManagerPlugin) Name() string {
	return metadata.Name
}

// Keyword returns the plugin's keyword.
func (p *PluginManagerPlugin) Keyword() string {
	return metadata.Keyword
}

// Init initializes the plugin.
func (p *PluginManagerPlugin) Init() tea.Cmd {
	return nil
}

// GetResults generates a sorted list of all plugins and their status.
func (p *PluginManagerPlugin) GetResults(query string) ([]plugin.Result, error) {
	mandatoryPlugins := []plugin.Result{}
	optionalPlugins := []plugin.Result{}

	// Process enabled plugins.
	loadedPlugins := p.mainPluginManager.GetAllPlugins()
	for kw, pl := range loadedPlugins {
		meta := pl.Metadata()
		result := plugin.Result{
			Title:       meta.Name,
			Description: fmt.Sprintf("Keyword: %s | Status: ✅ Enabled", kw),
			Identifier:  kw,
		}
		if meta.IsMandatory {
			mandatoryPlugins = append(mandatoryPlugins, result)
		} else {
			optionalPlugins = append(optionalPlugins, result)
		}
	}

	// Process disabled optional plugins.
	disabledPluginsMetadata := p.mainPluginManager.GetAllDisabledPluginsMetadatas()
	for kw, meta := range disabledPluginsMetadata {
		if _, exists := loadedPlugins[kw]; !exists { // Add only if not already listed as enabled.
			optionalPlugins = append(optionalPlugins, plugin.Result{
				Title:       meta.Name,
				Description: fmt.Sprintf("Keyword: %s | Status: ❌ Disabled (use --plugins=%s)", kw, meta.Flag),
				Identifier:  kw,
			})
		}
	}

	// Sort plugins alphabetically by Title.
	sort.Slice(mandatoryPlugins, func(i, j int) bool {
		return strings.ToLower(mandatoryPlugins[i].Title) < strings.ToLower(mandatoryPlugins[j].Title)
	})
	sort.Slice(optionalPlugins, func(i, j int) bool {
		return strings.ToLower(optionalPlugins[i].Title) < strings.ToLower(optionalPlugins[j].Title)
	})

	allResults := append(mandatoryPlugins, optionalPlugins...)

	// Add informational item about enabling plugins.
	allResults = append(allResults, plugin.Result{
		Title:       "Info",
		Description: "Use --plugins=flag1,flag2,... at startup to enable optional plugins.",
		Identifier:  "pm_info_flag",
	})

	// Filter results based on the query, excluding the info item from being filtered out.
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if trimmedQuery != "" {
		filteredResults := []plugin.Result{}
		for _, r := range allResults {
			if r.Identifier == "pm_info_flag" || // Always include the info item.
				strings.Contains(strings.ToLower(r.Title), trimmedQuery) ||
				strings.Contains(strings.ToLower(r.Description), trimmedQuery) {
				filteredResults = append(filteredResults, r)
			}
		}
		return filteredResults, nil
	}

	return allResults, nil
}

// Execute is a no-op for this plugin.
func (p *PluginManagerPlugin) Execute(identifier string) tea.Cmd {
	return nil
}

// Update is a no-op for this plugin.
func (p *PluginManagerPlugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	return p, nil
}

// View returns an empty string as this plugin uses the main application's list view.
func (p *PluginManagerPlugin) View() string {
	return ""
}

// GetError returns nil as this plugin does not maintain an error state.
func (p *PluginManagerPlugin) GetError() error {
	return nil
}
