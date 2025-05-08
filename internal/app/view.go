package app

import (
	"github.com/charmbracelet/lipgloss"
)

// View renders the application's UI.
func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("Exiting Incipio...")
	}

	var viewContent string
	activePlugin := m.pluginManager.GetCurrentPlugin()

	// Check if the active plugin provides a custom view.
	if activePlugin != nil {
		viewContent = activePlugin.View()
	}

	// Use the default list view if no plugin-specific view is provided.
	if viewContent == "" {
		viewContent = m.list.View()
	}

	// Combine the text input and the main content area (list or plugin view).
	mainContent := lipgloss.JoinVertical(lipgloss.Left,
		m.textInput.View(),
		viewContent,
	)

	// Apply the main application style.
	view := appStyle.Render(mainContent)

	return view
}
