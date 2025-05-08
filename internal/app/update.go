package app

import (
	"time"

	"github.com/barab-i/incipio/pkgs/plugin"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keyMap struct {
	Quit  key.Binding
	Esc   key.Binding
	Enter key.Binding
}

type clearSummaryMsg struct{}

type resultsMsg struct {
	results        []plugin.Result
	err            error
	pluginSwitched bool
	forQuery       string
}

const debounceDuration = 200 * time.Millisecond

type processQueryMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	queryBeforeInputUpdate := m.textInput.Value()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		textInputHeight := lipgloss.Height(m.textInput.View()) + 1
		listHeight := msg.Height - textInputHeight - appStyle.GetVerticalFrameSize()
		listHeight = max(1, listHeight)
		listWidth := msg.Width - appStyle.GetHorizontalFrameSize()
		m.list.SetSize(listWidth, listHeight)

		for _, pluginInstance := range m.pluginManager.plugins {
			if pluginInstance == nil {
				continue
			}
			updatedPlugin, pluginCmd := pluginInstance.Update(msg)
			m.updatePluginState(updatedPlugin)
			if pluginCmd != nil {
				cmds = append(cmds, pluginCmd)
			}
		}
		return m, tea.Batch(cmds...)

	case processQueryMsg:
		if m.debounceTimer != nil {
			queryCmd := m.handleQueryChange(m.lastQuery)
			if queryCmd != nil {
				cmds = append(cmds, queryCmd)
			}
		}
		m.debounceTimer = nil
		return m, tea.Batch(cmds...)

	case resultsMsg:
		if msg.forQuery != m.lastQuery {
			return m, nil // Stale results, ignore.
		}

		if msg.err != nil {
			m.err = msg.err
			m.list.SetItems([]list.Item{})
		} else {
			m.err = nil
			items := make([]list.Item, len(msg.results))
			for i, r := range msg.results {
				items[i] = listItem{
					title:       r.Title,
					description: r.Description,
					identifier:  r.Identifier,
				}
			}
			m.list.SetItems(items)
		}

		if msg.pluginSwitched {
			m.list.Select(0)
			m.list.ResetFilter()
		} else if len(m.list.Items()) > 0 {
			m.list.ResetSelected()
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Esc):
			if m.debounceTimer != nil {
				m.debounceTimer.Stop()
				m.debounceTimer = nil
			}
			if m.textInput.Value() != "" {
				m.textInput.SetValue("")
				m.err = nil
				queryCmd := m.handleQueryChange("")
				if queryCmd != nil {
					cmds = append(cmds, queryCmd)
				}

				if activePlugin := m.pluginManager.GetCurrentPlugin(); activePlugin != nil {
					updatedPlugin, pluginCmd := activePlugin.Update(clearSummaryMsg{})
					m.updatePluginState(updatedPlugin)
					if pluginCmd != nil {
						cmds = append(cmds, pluginCmd)
					}
				}
			} else {
				m.quitting = true
				return m, tea.Quit
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.Enter):
			if m.debounceTimer != nil {
				m.debounceTimer.Stop()
				m.debounceTimer = nil
				queryCmd := m.handleQueryChange(m.textInput.Value())
				if queryCmd != nil {
					cmds = append(cmds, queryCmd)
				}
			}
			if item := m.list.SelectedItem(); item != nil {
				if selectedItem, ok := item.(listItem); ok {
					execCmd := m.pluginManager.Execute(selectedItem.Identifier())
					// If Execute intends to quit, it should return tea.Quit.
					// The model's quitting flag is set if the command itself is tea.Quit.
					// This check is a basic way to see if the command is tea.Quit.
					// A more robust solution might involve specific return types or signals from Execute.
					if execCmd != nil && execCmd() == tea.Quit() {
						m.quitting = true
					}
					return m, execCmd
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Plugin Update Handling (for messages not handled by specific key matches)
	if activePlugin := m.pluginManager.GetCurrentPlugin(); activePlugin != nil {
		updatedPlugin, pluginCmd := activePlugin.Update(msg)
		m.updatePluginState(updatedPlugin)
		if pluginCmd != nil {
			cmds = append(cmds, pluginCmd)
		}
	}

	// Input Update
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	// Query Processing (Debounced)
	newQuery := m.textInput.Value()
	if newQuery != queryBeforeInputUpdate {
		m.lastQuery = newQuery
		if m.debounceTimer != nil {
			m.debounceTimer.Stop()
		}
		m.debounceTimer = time.NewTimer(debounceDuration)
		cmds = append(cmds, func() tea.Msg {
			if m.debounceTimer != nil {
				<-m.debounceTimer.C
				return processQueryMsg{}
			}
			return nil
		})
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) handleQueryChange(newQuery string) tea.Cmd {
	m.err = nil

	activePlugin, pluginSwitched := m.pluginManager.DetermineActivePlugin(newQuery)

	if pluginSwitched {
		m.list.SetItems([]list.Item{})
		m.list.ResetFilter()
	}

	if activePlugin == nil {
		if !pluginSwitched {
			m.list.SetItems([]list.Item{})
		}
		return nil
	}

	return func() tea.Msg {
		results, err := m.pluginManager.GetResults(newQuery)
		return resultsMsg{
			results:        results,
			err:            err,
			pluginSwitched: pluginSwitched,
			forQuery:       newQuery,
		}
	}
}

// updatePluginState delegates updating the plugin instance to the PluginManager.
func (m *model) updatePluginState(updatedPlugin plugin.Plugin) {
	if updatedPlugin == nil {
		return
	}
	m.pluginManager.UpdatePluginInstance(updatedPlugin)
}
