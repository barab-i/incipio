package calculator

import (
	"fmt"
	"strconv"

	"github.com/barab-i/incipio/pkgs/plugin"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/expr-lang/expr"
)

const Keyword = "="

var metadata = plugin.Metadata{
	Name:        "Calculator",
	Keyword:     Keyword,
	Flag:        "", // Mandatory, no flag needed.
	IsMandatory: true,
	IsDefault:   false,
}

// CalculatorPlugin implements the plugin.Plugin interface for calculations.
type CalculatorPlugin struct{}

// New creates a new instance of the CalculatorPlugin.
func New() *CalculatorPlugin {
	return &CalculatorPlugin{}
}

// Metadata returns the metadata for the plugin.
func (p *CalculatorPlugin) Metadata() plugin.Metadata {
	return metadata
}

// Name returns the plugin's name.
func (p *CalculatorPlugin) Name() string {
	return metadata.Name
}

// Keyword returns the plugin's keyword.
func (p *CalculatorPlugin) Keyword() string {
	return metadata.Keyword
}

// Init performs initial setup.
func (p *CalculatorPlugin) Init() tea.Cmd {
	return nil
}

// GetResults evaluates the mathematical expression in the query.
func (p *CalculatorPlugin) GetResults(query string) ([]plugin.Result, error) {
	if query == "" {
		return []plugin.Result{
			{
				Title:       "Calculator",
				Description: "Enter a mathematical expression after '=' (e.g., = 2 * (3 + 4))",
				Identifier:  "calc_info",
			},
		}, nil
	}

	program, err := expr.Compile(query)
	if err != nil {
		return []plugin.Result{
			{
				Title:       fmt.Sprintf("Error: %v", err),
				Description: "Invalid expression",
				Identifier:  "calc_error",
			},
		}, nil
	}

	result, err := expr.Run(program, nil)
	if err != nil {
		return []plugin.Result{
			{
				Title:       fmt.Sprintf("Error: %v", err),
				Description: "Evaluation failed",
				Identifier:  "calc_error",
			},
		}, nil
	}

	resultStr := formatResult(result)

	return []plugin.Result{
		{
			Title:       resultStr,
			Description: fmt.Sprintf("Result of: %s", query),
			Identifier:  resultStr,
		},
	}, nil
}

// formatResult converts the evaluation result into a string representation.
func formatResult(result any) string {
	switch v := result.(type) {
	case float64:
		// Check if the float is effectively an integer.
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return strconv.FormatFloat(v, 'f', -1, 64) // -1 for smallest number of digits.
	case int64:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", result) // Fallback for other types.
	}
}

// Execute handles the action for a selected result.
// For the calculator, it quits unless it's an info or error message.
func (p *CalculatorPlugin) Execute(identifier string) tea.Cmd {
	if identifier == "calc_info" || identifier == "calc_error" {
		return nil // Do nothing for info/error items.
	}
	return tea.Quit // Quit on selecting a valid result.
}

// Update handles messages.
func (p *CalculatorPlugin) Update(msg tea.Msg) (plugin.Plugin, tea.Cmd) {
	return p, nil // No-op command.
}

// View returns an empty string as this plugin uses the main application's list view.
func (p *CalculatorPlugin) View() string {
	return ""
}

// GetError returns nil as this plugin does not maintain an error state.
func (p *CalculatorPlugin) GetError() error {
	return nil
}
