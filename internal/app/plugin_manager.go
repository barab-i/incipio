package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/barab-i/incipio/pkgs/plugin"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

// PluginManager manages registered plugins.
type PluginManager struct {
	plugins                 map[string]plugin.Plugin
	disabledPluginsMetadata map[string]plugin.Metadata
	defaultPlugin           plugin.Plugin
	activePlugin            plugin.Plugin
	sortedKeywords          []string
}

// NewPluginManager creates a new PluginManager.
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins:                 make(map[string]plugin.Plugin),
		disabledPluginsMetadata: make(map[string]plugin.Metadata),
		sortedKeywords:          make([]string, 0),
	}
}

// RegisterPlugin adds an enabled plugin.
func (pm *PluginManager) RegisterPlugin(p plugin.Plugin) error {
	metadata := p.Metadata()
	keyword := metadata.Keyword

	if keyword == "" {
		return fmt.Errorf("plugin '%s' has an empty keyword", metadata.Name)
	}
	if _, exists := pm.plugins[keyword]; exists {
		return fmt.Errorf("plugin keyword '%s' from plugin '%s' is already registered", keyword, metadata.Name)
	}

	pm.plugins[keyword] = p
	zap.L().Info("Registered plugin",
		zap.String("name", metadata.Name),
		zap.String("keyword", keyword),
		zap.Bool("isDefault", metadata.IsDefault))

	if keyword != "" { // Default plugin might have an empty keyword conceptually, but metadata should have it.
		pm.sortedKeywords = append(pm.sortedKeywords, keyword)
		sort.Slice(pm.sortedKeywords, func(i, j int) bool {
			return len(pm.sortedKeywords[i]) > len(pm.sortedKeywords[j])
		})
	}

	if metadata.IsDefault {
		if pm.defaultPlugin != nil {
			zap.L().Warn("Overriding previous default plugin",
				zap.String("previousDefault", pm.defaultPlugin.Name()),
				zap.String("newDefault", metadata.Name))
		}
		pm.defaultPlugin = p
		zap.L().Info("Set default plugin", zap.String("name", metadata.Name))
		if pm.activePlugin == nil {
			pm.activePlugin = p
		}
	}
	return nil
}

// RegisterMetadata stores metadata for disabled plugins.
func (pm *PluginManager) RegisterMetadata(metadata plugin.Metadata) error {
	if metadata.Keyword == "" {
		return fmt.Errorf("plugin metadata '%s' has an empty keyword", metadata.Name)
	}
	if _, exists := pm.plugins[metadata.Keyword]; exists {
		// Plugin is already enabled and registered, no need to store metadata separately.
		return nil
	}
	if _, exists := pm.disabledPluginsMetadata[metadata.Keyword]; exists {
		// This is a notable condition, potentially indicating a configuration issue or duplicate plugin definition.
		zap.L().Warn("Plugin metadata keyword already registered as disabled",
			zap.String("keyword", metadata.Keyword),
			zap.String("pluginName", metadata.Name))
		// Continue to register, possibly overwriting, or return an error if strictness is desired.
		// For now, let's allow overwriting with a warning.
	}
	pm.disabledPluginsMetadata[metadata.Keyword] = metadata
	zap.L().Debug("Registered metadata for disabled plugin",
		zap.String("name", metadata.Name),
		zap.String("keyword", metadata.Keyword))
	return nil
}

// DetermineActivePlugin selects the active plugin based on the query.
func (pm *PluginManager) DetermineActivePlugin(query string) (plugin.Plugin, bool) {
	trimmedQuery := strings.TrimSpace(query)
	currentActiveKeyword := ""
	if pm.activePlugin != nil {
		currentActiveKeyword = pm.activePlugin.Keyword()
	}

	determinedPlugin := pm.defaultPlugin

	for _, keyword := range pm.sortedKeywords {
		if keyword != "" && strings.HasPrefix(trimmedQuery, keyword) {
			if len(trimmedQuery) == len(keyword) || (len(trimmedQuery) > len(keyword) && trimmedQuery[len(keyword)] == ' ') {
				if p, found := pm.plugins[keyword]; found {
					determinedPlugin = p
					break
				}
			}
		}
	}

	determinedKeyword := ""
	if determinedPlugin != nil {
		determinedKeyword = determinedPlugin.Keyword()
	}

	switched := currentActiveKeyword != determinedKeyword
	if switched {
		pm.activePlugin = determinedPlugin
	}

	if pm.activePlugin == nil { // Ensure a default is active if available
		pm.activePlugin = pm.defaultPlugin
	}
	return pm.activePlugin, switched
}

// GetCurrentPlugin returns the active plugin.
func (pm *PluginManager) GetCurrentPlugin() plugin.Plugin {
	if pm.activePlugin == nil {
		return pm.defaultPlugin
	}
	return pm.activePlugin
}

// GetResults retrieves results from the active plugin.
func (pm *PluginManager) GetResults(query string) ([]plugin.Result, error) {
	active := pm.GetCurrentPlugin()
	if active == nil {
		return nil, fmt.Errorf("no active plugin available to handle query")
	}

	pluginQuery := query
	trimmedQuery := strings.TrimSpace(query)
	activeKeyword := active.Keyword()

	if !active.Metadata().IsDefault && activeKeyword != "" && strings.HasPrefix(trimmedQuery, activeKeyword) {
		prefixLen := len(activeKeyword)
		if len(trimmedQuery) > prefixLen && trimmedQuery[prefixLen] == ' ' {
			pluginQuery = strings.TrimSpace(trimmedQuery[prefixLen+1:])
		} else if len(trimmedQuery) == prefixLen {
			pluginQuery = ""
		}
	}
	return active.GetResults(pluginQuery)
}

// Execute delegates execution to the active plugin.
func (pm *PluginManager) Execute(identifier string) tea.Cmd {
	active := pm.GetCurrentPlugin()
	if active == nil {
		zap.L().Warn("Execute called but no active plugin found", zap.String("identifier", identifier))
		return nil
	}
	return active.Execute(identifier)
}

// InitPlugins initializes all registered plugins.
func (pm *PluginManager) InitPlugins() tea.Cmd {
	var cmds []tea.Cmd
	initializedKeywords := make(map[string]bool)

	if pm.defaultPlugin != nil {
		keyword := pm.defaultPlugin.Keyword()
		if cmd := pm.defaultPlugin.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		if keyword != "" {
			initializedKeywords[keyword] = true
		}
	}

	for keyword, p := range pm.plugins {
		if _, alreadyInitialized := initializedKeywords[keyword]; !alreadyInitialized {
			if cmd := p.Init(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// GetAllPlugins returns all enabled plugins.
func (pm *PluginManager) GetAllPlugins() map[string]plugin.Plugin {
	return pm.plugins
}

// GetDefaultPlugin returns the default plugin.
func (pm *PluginManager) GetDefaultPlugin() plugin.Plugin {
	return pm.defaultPlugin
}

// GetAllDisabledPluginsMetadatas returns metadata for all disabled plugins.
func (pm *PluginManager) GetAllDisabledPluginsMetadatas() map[string]plugin.Metadata {
	return pm.disabledPluginsMetadata
}

// UpdatePluginInstance updates a registered plugin instance.
// It also updates the active and default plugin references if the updated plugin matches them.
func (pm *PluginManager) UpdatePluginInstance(updatedPlugin plugin.Plugin) {
	if updatedPlugin == nil {
		zap.L().Warn("Attempted to update with a nil plugin instance")
		return
	}

	keyword := updatedPlugin.Keyword()
	if keyword == "" {
		zap.L().Warn("Attempted to update plugin with an empty keyword", zap.String("pluginName", updatedPlugin.Name()))
		return
	}

	if _, exists := pm.plugins[keyword]; !exists {
		// This typically means an attempt to update a plugin that isn't registered.
		// Depending on desired behavior, this could be an error or a silent return.
		// For now, log a warning and return, as updating non-existent plugins is unexpected.
		zap.L().Warn("Attempted to update a plugin not currently registered",
			zap.String("keyword", keyword),
			zap.String("pluginName", updatedPlugin.Name()))
		return
	}

	pm.plugins[keyword] = updatedPlugin
	zap.L().Debug("Updated plugin instance in manager", zap.String("keyword", keyword), zap.String("pluginName", updatedPlugin.Name()))

	if pm.activePlugin != nil && pm.activePlugin.Keyword() == keyword {
		pm.activePlugin = updatedPlugin
		zap.L().Debug("Updated active plugin reference", zap.String("keyword", keyword))
	}

	if pm.defaultPlugin != nil && pm.defaultPlugin.Keyword() == keyword {
		pm.defaultPlugin = updatedPlugin
		zap.L().Debug("Updated default plugin reference", zap.String("keyword", keyword))
	}
}

// GetError delegates to the active plugin's GetError method.
// This function might be considered for removal if direct access is preferred,
// but it provides a consistent interface through the manager.
func (pm *PluginManager) GetError() error {
	active := pm.GetCurrentPlugin()
	if active == nil {
		return nil // Or an error indicating no active plugin
	}
	return active.GetError()
}
