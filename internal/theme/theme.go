package theme

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Theme defines the color scheme using Base16 names.
type Theme struct {
	Base00 lipgloss.Color `yaml:"base00"`
	Base01 lipgloss.Color `yaml:"base01"`
	Base02 lipgloss.Color `yaml:"base02"`
	Base03 lipgloss.Color `yaml:"base03"`
	Base04 lipgloss.Color `yaml:"base04"`
	Base05 lipgloss.Color `yaml:"base05"`
	Base06 lipgloss.Color `yaml:"base06"`
	Base07 lipgloss.Color `yaml:"base07"`
	Base08 lipgloss.Color `yaml:"base08"`
	Base09 lipgloss.Color `yaml:"base09"`
	Base0A lipgloss.Color `yaml:"base0a"`
	Base0B lipgloss.Color `yaml:"base0b"`
	Base0C lipgloss.Color `yaml:"base0c"`
	Base0D lipgloss.Color `yaml:"base0d"`
	Base0E lipgloss.Color `yaml:"base0e"`
	Base0F lipgloss.Color `yaml:"base0f"`
}

// DefaultTheme provides a default Base16-like theme.
var DefaultTheme = Theme{ // Base16 Catppuccin Mocha
	Base00: lipgloss.Color("#1e1e2e"),
	Base01: lipgloss.Color("#181825"),
	Base02: lipgloss.Color("#313244"),
	Base03: lipgloss.Color("#45475a"),
	Base04: lipgloss.Color("#585b70"),
	Base05: lipgloss.Color("#cdd6f4"),
	Base06: lipgloss.Color("#f5e0dc"),
	Base07: lipgloss.Color("#b4befe"),
	Base08: lipgloss.Color("#f38ba8"),
	Base09: lipgloss.Color("#fab387"),
	Base0A: lipgloss.Color("#f9e2af"),
	Base0B: lipgloss.Color("#a6e3a1"),
	Base0C: lipgloss.Color("#94e2d5"),
	Base0D: lipgloss.Color("#89b4fa"),
	Base0E: lipgloss.Color("#cba6f7"),
	Base0F: lipgloss.Color("#f2cdcd"),
}

// CurrentTheme holds the active theme. Initially set to DefaultTheme.
var CurrentTheme = DefaultTheme

const configFileName = "theme.yaml"
const configDir = "incipio"

// LoadThemeFromFile attempts to load theme colors from a YAML config file.
// If loading fails or the file doesn't exist, it falls back to DefaultTheme.
func LoadThemeFromFile() {
	configPath, err := xdg.ConfigFile(filepath.Join(configDir, configFileName))
	if err != nil {
		zap.L().Warn("Could not determine theme config path, using default theme.", zap.Error(err))
		CurrentTheme = DefaultTheme
		return
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		zap.L().Info("Theme config file not found, using default theme.", zap.String("path", configPath))
		CurrentTheme = DefaultTheme
		return
	}

	yamlFileBytes, err := os.ReadFile(configPath)
	if err != nil {
		zap.L().Warn("Error reading theme config file, using default theme.", zap.String("path", configPath), zap.Error(err))
		CurrentTheme = DefaultTheme
		return
	}

	lowerYamlContent := strings.ToLower(string(yamlFileBytes))
	rawThemeData := make(map[string]string)

	err = yaml.Unmarshal([]byte(lowerYamlContent), &rawThemeData)
	if err != nil {
		zap.L().Warn("Error unmarshalling theme config YAML, using default theme.", zap.String("path", configPath), zap.Error(err))
		CurrentTheme = DefaultTheme
		return
	}

	getColor := func(lowerKey string, defaultValue lipgloss.Color) lipgloss.Color {
		val, ok := rawThemeData[lowerKey]
		if !ok || val == "" {
			return defaultValue
		}

		if !strings.HasPrefix(val, "#") {
			val = "#" + val
		}
		// Basic hex format validation
		if len(val) != 7 {
			zap.L().Warn("Invalid hex color format in theme config, using default for key.",
				zap.String("key", lowerKey),
				zap.String("value", val),
				zap.String("path", configPath))
			return defaultValue
		}
		// Further validation could be added here if lipgloss.Color doesn't handle invalid hex gracefully.
		return lipgloss.Color(val)
	}

	CurrentTheme = Theme{
		Base00: getColor("base00", DefaultTheme.Base00),
		Base01: getColor("base01", DefaultTheme.Base01),
		Base02: getColor("base02", DefaultTheme.Base02),
		Base03: getColor("base03", DefaultTheme.Base03),
		Base04: getColor("base04", DefaultTheme.Base04),
		Base05: getColor("base05", DefaultTheme.Base05),
		Base06: getColor("base06", DefaultTheme.Base06),
		Base07: getColor("base07", DefaultTheme.Base07),
		Base08: getColor("base08", DefaultTheme.Base08),
		Base09: getColor("base09", DefaultTheme.Base09),
		Base0A: getColor("base0a", DefaultTheme.Base0A),
		Base0B: getColor("base0b", DefaultTheme.Base0B),
		Base0C: getColor("base0c", DefaultTheme.Base0C),
		Base0D: getColor("base0d", DefaultTheme.Base0D),
		Base0E: getColor("base0e", DefaultTheme.Base0E),
		Base0F: getColor("base0f", DefaultTheme.Base0F),
	}

	zap.L().Info("Theme loaded from config file.", zap.String("path", configPath))
}
