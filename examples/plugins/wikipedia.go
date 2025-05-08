package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/barab-i/incipio/internal/theme"
	"github.com/barab-i/incipio/pkgs/plugin"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	wikipediaAPI = "https://en.wikipedia.org/w/api.php" // Wikipedia API base URL.
	keyword      = "!w"                                 // Plugin activation keyword.
	userAgent    = "incipio-launcher/0.1"               // User-Agent for Wikipedia API requests.
)

// metadata defines Wikipedia plugin properties.
var metadata = plugin.Metadata{
	Name:        "Wikipedia Search",
	Description: "Search Wikipedia articles and view summaries.",
	Keyword:     keyword,
	Flag:        "wikipedia",
}

// API response structures
type openSearchResponse []any // [query, [titles], [descriptions], [urls]]
type queryResponse struct {
	Query struct {
		Pages map[string]struct {
			Extract string `json:"extract"` // Introductory text of the page.
		} `json:"pages"`
	} `json:"query"`
}

// Messages
type summaryFetchedMsg struct { // Sent when a Wikipedia article summary is fetched.
	content string
	err     error
}

type clearSummaryMsg struct{} // Sent by the main app to clear the plugin's view.

// KeyMap
type viewportKeyMap struct { // Defines viewport navigation keybindings.
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Down         key.Binding
	Up           key.Binding
}

var defaultViewportKeys = viewportKeyMap{ // Default viewport keybindings.
	PageDown:     key.NewBinding(key.WithKeys("pgdown", " ", "f"), key.WithHelp("pgdn", "page down")),
	PageUp:       key.NewBinding(key.WithKeys("pgup", "b"), key.WithHelp("pgup", "page up")),
	HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up")),
	HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down")),
	Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
}

// Base Styles (Theme applied in New)
var (
	baseTitleStyle = func() lipgloss.Style { // Base style for the title header.
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()
	baseInfoStyle = func() lipgloss.Style { // Base style for the info footer.
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()
	baseLineStyle  = lipgloss.NewStyle().MarginTop(1) // Base style for separator lines.
	baseErrorStyle = lipgloss.NewStyle().MarginTop(1) // Base style for error messages.
)

// WikipediaPlugin handles Wikipedia search and summary display.
type WikipediaPlugin struct {
	httpClient       *http.Client
	currentPageTitle string
	currentSummary   string
	isLoading        bool
	viewWidth        int
	viewHeight       int
	viewport         viewport.Model
	keys             viewportKeyMap
	ready            bool // True if viewport dimensions are set.
	err              error

	// Theme-aware styles, initialized in New().
	titleStyle lipgloss.Style
	infoStyle  lipgloss.Style
	lineStyle  lipgloss.Style
	errorStyle lipgloss.Style
}

// New creates a WikipediaPlugin.
func New() plugin.Plugin {
	vp := viewport.New(0, 0)       // Dimensions set later via WindowSizeMsg.
	vp.Style = lipgloss.NewStyle() // Default style; container handles padding.

	p := &WikipediaPlugin{
		httpClient: &http.Client{},
		viewport:   vp,
		keys:       defaultViewportKeys,
		// Init with base styles; theme applied next.
		titleStyle: baseTitleStyle,
		infoStyle:  baseInfoStyle,
		lineStyle:  baseLineStyle,
		errorStyle: baseErrorStyle,
	}

	// Apply theme colors.
	p.titleStyle = p.titleStyle.BorderForeground(lipgloss.Color(theme.CurrentTheme.Base0D))
	p.infoStyle = p.infoStyle.BorderForeground(lipgloss.Color(theme.CurrentTheme.Base0D))
	p.lineStyle = p.lineStyle.Foreground(lipgloss.Color(theme.CurrentTheme.Base0D))
	p.errorStyle = p.errorStyle.Foreground(lipgloss.Color(theme.CurrentTheme.Base08))

	return p
}

// Metadata returns static plugin metadata.
func (p *WikipediaPlugin) Metadata() plugin.Metadata {
	return metadata
}

// Name returns plugin display name.
func (p *WikipediaPlugin) Name() string {
	return metadata.Name
}

// Keyword returns plugin activation keyword.
func (p *WikipediaPlugin) Keyword() string {
	return metadata.Keyword
}

// Init resets plugin state (called on load/re-init).
func (p *WikipediaPlugin) Init() tea.Cmd {
	p.resetState()
	p.ready = false                      // Not ready until WindowSizeMsg.
	return func() tea.Msg { return nil } // No-op tea.Cmd.
}

// resetState clears dynamic data (summary, title, error, loading status).
func (p *WikipediaPlugin) resetState() {
	p.currentPageTitle = ""
	p.currentSummary = ""
	p.isLoading = false
	p.err = nil
	if p.ready { // Avoid panic if viewport not initialized.
		p.viewport.SetContent("")
		p.viewport.YOffset = 0
	}
}

// GetResults searches Wikipedia using OpenSearch API.
// Returns results or an error item for display.
func (p *WikipediaPlugin) GetResults(query string) ([]plugin.Result, error) {
	p.resetState() // Clear previous view state first.

	if query == "" {
		return []plugin.Result{
			{
				Title:       "Wikipedia Search",
				Description: "Enter a search term (e.g., !w Golang)",
				Identifier:  "wiki_info", // Informational message identifier.
			},
		}, nil
	}

	params := url.Values{}
	params.Add("action", "opensearch")
	params.Add("search", query)
	params.Add("limit", "10")    // Max results.
	params.Add("namespace", "0") // Main article namespace.
	params.Add("format", "json")
	requestURL := wikipediaAPI + "?" + params.Encode()

	respBody, err := p.doAPIRequest(requestURL, "opensearch")
	if err != nil {
		// Return API error as a displayable result.
		return []plugin.Result{
			{Title: "Wikipedia API Error", Description: err.Error(), Identifier: "wiki_api_error"},
		}, nil // Error info is in the result.
	}

	var apiResponse openSearchResponse
	if err := json.Unmarshal(respBody, &apiResponse); err != nil {
		err = fmt.Errorf("failed to parse Wikipedia opensearch response: %w", err)
		return []plugin.Result{
			{Title: "Wikipedia Parse Error", Description: err.Error(), Identifier: "wiki_parse_error"},
		}, nil
	}

	// Validate OpenSearch API response structure.
	if len(apiResponse) != 4 {
		err = fmt.Errorf("unexpected Wikipedia opensearch API response format")
		return []plugin.Result{
			{Title: "Wikipedia API Error", Description: err.Error(), Identifier: "wiki_format_error"},
		}, nil
	}
	titles, okT := apiResponse[1].([]any)
	descriptions, okD := apiResponse[2].([]any)
	urls, okU := apiResponse[3].([]any) // URLs checked for response integrity.
	if !okT || !okD || !okU || len(titles) != len(descriptions) || len(titles) != len(urls) {
		err = fmt.Errorf("invalid data structure in Wikipedia opensearch response")
		return []plugin.Result{
			{Title: "Wikipedia API Error", Description: err.Error(), Identifier: "wiki_structure_error"},
		}, nil
	}

	results := make([]plugin.Result, 0, len(titles))
	for i := range titles {
		title, okTitle := titles[i].(string)
		description, okDesc := descriptions[i].(string)
		if !okTitle || !okDesc {
			// Skip items with unexpected types.
			continue
		}
		results = append(results, plugin.Result{
			Title:       title,
			Description: description,
			Identifier:  title, // Page title is identifier for Execute.
		})
	}

	if len(results) == 0 {
		return []plugin.Result{
			{
				Title:       fmt.Sprintf("No results found for '%s'", query),
				Description: "Try a different search term.",
				Identifier:  "wiki_no_results",
			},
		}, nil
	}

	return results, nil
}

// Execute fetches the summary for the selected article (identified by page title).
// Returns a tea.Cmd for asynchronous API request.
func (p *WikipediaPlugin) Execute(identifier string) tea.Cmd {
	// Ignore execution for informational or error results.
	if identifier == "wiki_info" || identifier == "wiki_no_results" || strings.HasPrefix(identifier, "wiki_") {
		p.resetState()                       // Clear view if it was an info/error item.
		return func() tea.Msg { return nil } // No-op command.
	}

	pageTitle := identifier
	p.resetState() // Clear state before loading new summary.
	p.currentPageTitle = pageTitle
	p.isLoading = true
	p.updateViewportContent() // Show loading indicator in viewport.

	// Return command for Bubble Tea runtime.
	return func() tea.Msg {
		params := url.Values{}
		params.Add("action", "query")
		params.Add("format", "json")
		params.Add("titles", pageTitle)
		params.Add("prop", "extracts")    // Request page extracts.
		params.Add("exintro", "true")     // Introductory section only.
		params.Add("explaintext", "true") // Plain text, not HTML.
		params.Add("redirects", "1")      // Follow redirects.
		requestURL := wikipediaAPI + "?" + params.Encode()

		respBody, err := p.doAPIRequest(requestURL, "fetch-extract")
		if err != nil {
			return summaryFetchedMsg{err: err} // Return error in message.
		}

		var queryResp queryResponse
		if err := json.Unmarshal(respBody, &queryResp); err != nil {
			return summaryFetchedMsg{err: fmt.Errorf("failed to parse Wikipedia query response: %w", err)}
		}

		var extract string
		// API returns map of pages by ID; iterate to find extract.
		// Expect one page in response.
		for _, page := range queryResp.Query.Pages {
			extract = page.Extract
			break
		}

		if extract == "" {
			// Handle page existing but no intro text.
			extract = fmt.Sprintf("No summary found for '%s'. The page might exist but have no introductory text.", pageTitle)
		}

		return summaryFetchedMsg{content: extract}
	}
}

// Update handles messages (fetched summaries, window size changes, etc.).
// Updates plugin state and viewport.
func (p *WikipediaPlugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd // For multiple commands.

	switch msg := msg.(type) {
	case summaryFetchedMsg:
		p.isLoading = false
		if msg.err != nil {
			p.err = msg.err
			p.currentSummary = "" // Clear summary on error.
		} else {
			p.currentSummary = msg.content
			p.err = nil // Clear previous error.
		}
		p.updateViewportContent()               // Update viewport with new content/error.
		p.viewport.YOffset = 0                  // Reset scroll for new content.
		return p, func() tea.Msg { return nil } // No-op command.

	case clearSummaryMsg:
		p.resetState()                          // Clear plugin's view and state.
		return p, func() tea.Msg { return nil } // No-op command.

	case tea.WindowSizeMsg:
		// Constants for main app layout estimation.
		const mainAppHorizontalPadding = 4
		const mainAppVerticalPadding = 2
		const textInputHeight = 1 // Estimated text input height.

		// Calculate available width/height for plugin view.
		p.viewWidth = msg.Width - mainAppHorizontalPadding
		p.viewHeight = msg.Height - textInputHeight - mainAppVerticalPadding
		p.viewHeight = max(1, p.viewHeight) // Min height 1.

		// Calculate viewport dimensions (accounts for header/footer).
		headerHeight := lipgloss.Height(p.headerView())
		footerHeight := lipgloss.Height(p.footerView())
		vpWidth := p.viewWidth
		vpHeight := p.viewHeight - headerHeight - footerHeight

		// Ensure positive viewport dimensions.
		p.viewport.Width = max(1, vpWidth)
		p.viewport.Height = max(1, vpHeight)
		p.viewport.YPosition = headerHeight // Position below header.

		p.ready = true                          // Plugin ready with dimensions.
		p.updateViewportContent()               // Re-wrap content.
		return p, func() tea.Msg { return nil } // No-op command.

	case tea.KeyMsg:
		// Key messages handled by viewport.Update below.
		break
	}

	// Process viewport updates for relevant messages (e.g., KeyMsg).
	if p.ready {
		// Only update viewport if content is visible (not loading, no error, summary present).
		if !p.isLoading && p.err == nil && p.currentSummary != "" {
			p.viewport, cmd = p.viewport.Update(msg) // Pass original msg.
			cmds = append(cmds, cmd)
		}
	}

	return p, tea.Batch(cmds...)
}

// updateViewportContent sets viewport content (loading, error, or summary).
// Handles text wrapping.
func (p *WikipediaPlugin) updateViewportContent() {
	if !p.ready {
		// Don't update if viewport dimensions not set.
		return
	}

	contentWidth := p.viewport.Width // Use actual viewport width for wrapping.
	var content string

	switch {
	case p.isLoading:
		content = "Loading summary..."
	case p.err != nil:
		// Use theme-aware errorStyle.
		wrappedError := p.errorStyle.Width(contentWidth).Render(fmt.Sprintf("Error: %v", p.err))
		content = wrappedError
	case p.currentSummary != "":
		// Wrap summary text.
		wrappingStyle := lipgloss.NewStyle().Width(contentWidth)
		wrappedContent := wrappingStyle.Render(p.currentSummary)
		content = wrappedContent
	default:
		// Clear content if no other state applies.
		content = ""
	}

	p.viewport.SetContent(content)
}

// headerView renders the title bar.
// Displays current article title or default, styled with theme.
func (p *WikipediaPlugin) headerView() string {
	titleStr := "Wikipedia" // Default.
	if p.currentPageTitle != "" {
		titleStr = p.currentPageTitle
	}

	// Use theme-aware styles.
	currentTitleStyle := p.titleStyle
	currentLineStyle := p.lineStyle

	// Calculate available width for title text (excluding border/padding).
	titleStyleBaseWidth := currentTitleStyle.GetHorizontalFrameSize()
	maxTextWidth := p.viewWidth - titleStyleBaseWidth
	maxTextWidth = max(0, maxTextWidth) // Ensure non-negative.

	// Truncate title if it exceeds available width.
	if lipgloss.Width(titleStr) > maxTextWidth {
		titleStr = truncateString(titleStr, maxTextWidth)
	}

	title := currentTitleStyle.Render(titleStr)
	titleRenderedWidth := lipgloss.Width(title)

	// Fill remaining header space with a line.
	lineLength := max(0, p.viewWidth-titleRenderedWidth)
	line := currentLineStyle.Render(strings.Repeat("─", lineLength))

	return lipgloss.JoinHorizontal(lipgloss.Top, title, line)
}

// footerView renders the footer.
// Displays scroll percentage, styled with theme.
func (p *WikipediaPlugin) footerView() string {
	infoStr := "---%" // Default if viewport not ready/no height.
	if p.ready && p.viewport.Height > 0 {
		infoStr = fmt.Sprintf("%3.f%%", p.viewport.ScrollPercent()*100)
	}

	// Use theme-aware styles.
	currentInfoStyle := p.infoStyle
	currentLineStyle := p.lineStyle

	info := currentInfoStyle.Render(infoStr)
	infoRenderedWidth := lipgloss.Width(info)

	// Fill space to the left of info with a line.
	lineLength := max(0, p.viewWidth-infoRenderedWidth)
	line := currentLineStyle.Render(strings.Repeat("─", lineLength))

	return lipgloss.JoinHorizontal(lipgloss.Top, line, info)
}

// View renders the complete plugin UI (header, viewport, footer).
// Returns empty string if not ready or no content.
func (p *WikipediaPlugin) View() string {
	// Don't render if not ready or no active state.
	if !p.ready || (!p.isLoading && p.err == nil && p.currentSummary == "" && p.currentPageTitle == "") {
		return ""
	}

	// Viewport content is already set/wrapped by updateViewportContent.
	fullView := lipgloss.JoinVertical(lipgloss.Left,
		p.headerView(),
		p.viewport.View(), // Render viewport.
		p.footerView(),
	)

	// Constrain final output to allocated view dimensions.
	containerStyle := lipgloss.NewStyle().Width(p.viewWidth).Height(p.viewHeight)

	return containerStyle.Render(fullView)
}

// GetError returns any plugin error.
func (p *WikipediaPlugin) GetError() error {
	return p.err
}

// Helper Functions

// doAPIRequest performs HTTP GET to Wikipedia API.
// Includes User-Agent and handles common errors.
// 'operation' string aids error messages.
func (p *WikipediaPlugin) doAPIRequest(requestURL, operation string) ([]byte, error) {
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Wikipedia request (%s): %w", operation, err)
	}
	// Set User-Agent (Wikipedia API guideline).
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s)", userAgent, operation))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Wikipedia (%s): %w", operation, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Ignore ReadAll error, primary error is status code.
		bodyStr := strings.TrimSpace(string(bodyBytes))
		if bodyStr != "" {
			return nil, fmt.Errorf("wikipedia API error (%s): status %s - %s", operation, resp.Status, bodyStr)
		}
		return nil, fmt.Errorf("wikipedia API error (%s): status %s", operation, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Wikipedia response body (%s): %w", operation, err)
	}
	return body, nil
}

// truncateString shortens a string to a maximum width, adding "..." if truncated.
// Operates on runes for multi-byte character handling.
func truncateString(s string, maxWidth int) string {
	if maxWidth <= 0 { // Handle edge case.
		return ""
	}

	runes := []rune(s)
	runeLen := len(runes)

	if runeLen <= maxWidth { // String already fits.
		return s
	}

	if maxWidth <= 3 { // Not enough space for ellipsis.
		return string(runes[:maxWidth])
	}

	return string(runes[:maxWidth-3]) + "..." // Truncate and add ellipsis.
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
