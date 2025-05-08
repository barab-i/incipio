package main

import (
	"fmt"

	"github.com/barab-i/incipio/pkgs/plugin"
	tea "github.com/charmbracelet/bubbletea"
)

// helloMetadata contains the metadata for the HelloPlugin.
var helloMetadata = plugin.Metadata{
	Name:        "Hello",
	Description: "A simple example plugin that greets the user.",
	Keyword:     "!hello",
	Flag:        "hello", // Command-line flag to enable this optional plugin.
}

// HelloPlugin is an example plugin.
type HelloPlugin struct{}

// New creates a new instance of HelloPlugin.
// This function is required by Yaegi to instantiate the plugin.
func New() plugin.Plugin {
	return &HelloPlugin{}
}

// Metadata returns the plugin's metadata.
func (p *HelloPlugin) Metadata() plugin.Metadata { return helloMetadata }

// Name returns the plugin's display name.
func (p *HelloPlugin) Name() string { return helloMetadata.Name }

// Keyword returns the keyword used to activate this plugin.
func (p *HelloPlugin) Keyword() string { return helloMetadata.Keyword }

// Init initializes the plugin.
// It returns a no-op command as this plugin requires no initial actions.
func (p *HelloPlugin) Init() tea.Cmd {
	return func() tea.Msg { return nil } // Return a no-op command.
}

// GetResults returns a list of results based on the query.
func (p *HelloPlugin) GetResults(query string) ([]plugin.Result, error) {
	return []plugin.Result{
		{
			Title:       "Hello!",
			Description: fmt.Sprintf("You typed: %s", query),
			Identifier:  "hello_result",
		},
	}, nil
}

// Execute is called when a result is selected.
// This plugin quits the application upon selection.
func (p *HelloPlugin) Execute(identifier string) tea.Cmd {
	return tea.Quit
}

// Update handles incoming Bubble Tea messages.
// It returns the plugin and a no-op command as this plugin doesn't change state based on messages.
func (p *HelloPlugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	return p, func() tea.Msg { return nil } // Return a no-op command.
}

// View returns an empty string as this plugin uses the main application's list view.
func (p *HelloPlugin) View() string { return "" }

// GetError returns nil as this plugin does not maintain a persistent error state.
func (p *HelloPlugin) GetError() error { return nil }
