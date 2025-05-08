package main

import (
	"flag"
	"log"
	"strings"

	"github.com/barab-i/incipio/internal/app"
	"github.com/barab-i/incipio/internal/plugins/applauncher"
	"github.com/barab-i/incipio/internal/plugins/calculator"
	"github.com/barab-i/incipio/internal/plugins/pluginmanager"
	"github.com/barab-i/incipio/internal/theme"
	"github.com/barab-i/incipio/internal/yaegi"
	"github.com/barab-i/incipio/pkgs/plugin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	enabledPluginsFlag = flag.String("plugins", "", "Comma-separated list of optional plugins to enable.")
	debugFlag          = flag.Bool("debug", false, "Enable debug logging.")
)

func main() {
	flag.Parse()

	logger := initializeLogger(*debugFlag)
	defer logger.Sync()

	theme.LoadThemeFromFile()
	app.InitStyles()

	pluginManager := app.NewPluginManager()
	registerPlugins(pluginManager, logger)

	initialModel := app.InitialModel(pluginManager)
	runProgram(initialModel, logger)
}

func initializeLogger(debug bool) *zap.Logger {
	var config zap.Config
	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.Level.SetLevel(zapcore.WarnLevel)
	}

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	zap.ReplaceGlobals(logger)

	if debug {
		logger.Info("Debug mode enabled. Using development logger.")
	}
	return logger
}

func registerPlugins(pluginManager *app.PluginManager, logger *zap.Logger) {
	builtInPlugins := []plugin.Plugin{
		applauncher.New(),
		calculator.New(),
		pluginmanager.New(pluginManager),
	}

	yaegiPlugins, err := yaegi.LoadPlugins()
	if err != nil {
		logger.Warn("Could not load Yaegi plugins", zap.Error(err))
	}

	allPlugins := append(builtInPlugins, yaegiPlugins...)
	enabledOptionalPlugins := parseEnabledPlugins(*enabledPluginsFlag)

	for _, p := range allPlugins {
		metadata := p.Metadata()
		_, isEnabled := enabledOptionalPlugins[metadata.Flag]
		shouldRegister := metadata.IsMandatory || isEnabled

		if shouldRegister {
			if err := pluginManager.RegisterPlugin(p); err != nil {
				logger.Fatal("Error registering plugin", zap.String("pluginName", p.Name()), zap.Error(err))
			}
		} else if !metadata.IsMandatory {
			if err := pluginManager.RegisterMetadata(metadata); err != nil {
				logger.Warn("Could not register metadata for plugin", zap.String("pluginName", metadata.Name), zap.Error(err))
			}
		}
	}
}

func parseEnabledPlugins(flagValue string) map[string]struct{} {
	enabledPlugins := make(map[string]struct{})
	if flagValue != "" {
		for f := range strings.SplitSeq(flagValue, ",") {
			trimmedFlag := strings.TrimSpace(f)
			if trimmedFlag != "" {
				enabledPlugins[trimmedFlag] = struct{}{}
			}
		}
	}
	return enabledPlugins
}

func runProgram(initialModel tea.Model, logger *zap.Logger) {
	program := tea.NewProgram(initialModel, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		logger.Fatal("Error running program", zap.Error(err))
	}
}
