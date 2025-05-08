package app

import (
	"fmt"
	"io"
	"time"

	"github.com/barab-i/incipio/internal/theme"
	"go.uber.org/zap"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle          lipgloss.Style
	listTitleStyle    lipgloss.Style
	itemStyle         lipgloss.Style
	selectedItemStyle lipgloss.Style
	descStyle         lipgloss.Style
	paginationStyle   lipgloss.Style
	helpStyle         lipgloss.Style
	inputPromptStyle  lipgloss.Style
	inputTextStyle    lipgloss.Style
	quitTextStyle     lipgloss.Style
)

// InitStyles initializes styles using the current theme.
func InitStyles() {
	appStyle = lipgloss.NewStyle().Padding(1, 2)
	listTitleStyle = lipgloss.NewStyle().
		MarginLeft(0).
		Padding(0, 1).
		Background(theme.CurrentTheme.Base0D).
		Foreground(theme.CurrentTheme.Base00)

	itemStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(theme.CurrentTheme.Base05)

	selectedItemStyle = lipgloss.NewStyle().
		PaddingLeft(0).
		Foreground(theme.CurrentTheme.Base0E).
		SetString("> ")

	descStyle = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(theme.CurrentTheme.Base03)

	paginationStyle = list.DefaultStyles().PaginationStyle.
		PaddingLeft(2).
		Foreground(theme.CurrentTheme.Base04)

	helpStyle = list.DefaultStyles().HelpStyle.
		PaddingLeft(2).
		PaddingBottom(1).
		Foreground(theme.CurrentTheme.Base04)

	inputPromptStyle = lipgloss.NewStyle().
		Foreground(theme.CurrentTheme.Base0A).
		Bold(true)

	inputTextStyle = lipgloss.NewStyle().
		Foreground(theme.CurrentTheme.Base05)

	quitTextStyle = lipgloss.NewStyle().
		Margin(1, 0, 2, 4).
		Foreground(theme.CurrentTheme.Base08)
}

// KeyMap defines the keybindings for the application.
type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
	Esc   key.Binding
}

// DefaultKeyMap provides the default keybindings.
var DefaultKeyMap = KeyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move up")),
	Down:  key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move down")),
	Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Quit:  key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	Esc:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("escape", "clear/quit")),
}

// listItem adapts plugin.Result to the list.Item interface.
type listItem struct {
	title       string
	description string
	identifier  string
}

func (i listItem) FilterValue() string { return i.title }
func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.description }
func (i listItem) Identifier() string  { return i.identifier }

// itemDelegate provides custom rendering for list items.
type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	li, ok := item.(listItem)
	if !ok {
		return
	}

	var titleRendered, descRendered, combined string
	separator := " " // Separator between title and description

	descRendered = descStyle.Render(li.Description())

	if index == m.Index() {
		titleRendered = selectedItemStyle.Render(li.Title())
		combined = lipgloss.JoinHorizontal(lipgloss.Left, titleRendered, descRendered)
	} else {
		titleRendered = itemStyle.Render(li.Title())
		combined = lipgloss.JoinHorizontal(lipgloss.Left, titleRendered, separator, descRendered)
		combined = itemStyle.Render(combined)
	}

	fmt.Fprint(w, combined)
}

// model holds the application's state.
type model struct {
	pluginManager *PluginManager
	list          list.Model
	textInput     textinput.Model
	keys          KeyMap
	width         int
	height        int
	err           error // err stores an error to be displayed in the UI.
	quitting      bool

	debounceTimer *time.Timer // For debouncing query processing.
	lastQuery     string      // Stores the query for the debounced call.
}

// InitialModel sets up the initial state of the application.
func InitialModel(pm *PluginManager) model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50 // Initial width, will be updated.
	ti.Prompt = "> "
	ti.PromptStyle = inputPromptStyle
	ti.TextStyle = inputTextStyle

	delegate := itemDelegate{}

	li := list.New([]list.Item{}, delegate, 0, 0)
	li.Title = "" // No global title for the list itself.
	li.Styles.Title = lipgloss.NewStyle().MarginLeft(0).Padding(0, 1).Foreground(theme.CurrentTheme.Base0D)
	li.Styles.PaginationStyle = paginationStyle
	li.Styles.HelpStyle = helpStyle

	li.SetShowHelp(false)
	li.SetShowStatusBar(false)
	li.SetFilteringEnabled(false) // Plugins handle their own filtering logic.
	li.SetShowFilter(false)

	li.KeyMap = list.KeyMap{
		CursorUp:   DefaultKeyMap.Up,
		CursorDown: DefaultKeyMap.Down,
		GoToStart:  key.NewBinding(key.WithKeys("home")),
		GoToEnd:    key.NewBinding(key.WithKeys("end")),
	}

	m := model{
		pluginManager: pm,
		textInput:     ti,
		list:          li,
		keys:          DefaultKeyMap,
		err:           nil,
	}

	// Fetch initial items from the default plugin.
	// Note: The tea.Cmd returned by m.pluginManager.InitPlugins() is currently ignored here.
	// Proper handling would involve returning it from model.Init() and processing it in the Bubble Tea runtime.
	m.pluginManager.InitPlugins() // Synchronous part of plugin initialization.

	defaultPlugin := m.pluginManager.GetCurrentPlugin()
	if defaultPlugin != nil {
		results, err := defaultPlugin.GetResults("") // Fetch initial results.
		if err != nil {
			// Store error for UI display and log it.
			m.err = fmt.Errorf("initial load failed for plugin '%s': %w", defaultPlugin.Name(), err)
			zap.L().Error("Initial results load failed for default plugin",
				zap.String("plugin", defaultPlugin.Name()),
				zap.Error(err)) // Log the original error.
			m.list.SetItems([]list.Item{}) // Keep the list empty on error.
		} else {
			// Convert plugin results to list items.
			items := make([]list.Item, len(results))
			for i, r := range results {
				items[i] = listItem{
					title:       r.Title,
					description: r.Description,
					identifier:  r.Identifier,
				}
			}
			m.list.SetItems(items) // Populate the list with initial items.
		}
	} else {
		m.list.SetItems([]list.Item{}) // Ensure list is empty if no default plugin is available.
		zap.L().Warn("No default plugin found during initial model setup.")
	}

	return m
}

// Init performs initial setup for the model, like starting the text input blink.
// Note: This should ideally also return commands from plugin initialization (see InitialModel).
func (m model) Init() tea.Cmd {
	return textinput.Blink
}
