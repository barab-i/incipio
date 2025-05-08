package yaegi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/barab-i/incipio/pkgs/plugin"
	"github.com/barab-i/incipio/pkgs/symbol"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"go.uber.org/zap"
)

const PluginDirName = "incipio/plugins"

// LoadPlugins scans the plugin directory and loads Go plugins using Yaegi.
func LoadPlugins() ([]plugin.Plugin, error) {
	pluginDirPath := filepath.Join(xdg.ConfigHome, PluginDirName)

	if _, err := os.Stat(xdg.ConfigHome); os.IsNotExist(err) {
		zap.L().Info("XDG config home directory does not exist, Yaegi plugins cannot be loaded yet.", zap.String("path", xdg.ConfigHome))
		return nil, nil
	}

	if _, err := os.Stat(pluginDirPath); os.IsNotExist(err) {
		zap.L().Info("Yaegi plugin directory not found, skipping plugin loading.", zap.String("path", pluginDirPath))
		return nil, nil
	}

	files, err := os.ReadDir(pluginDirPath)
	if err != nil {
		return nil, fmt.Errorf("could not read yaegi plugin directory '%s': %w", pluginDirPath, err)
	}

	var loadedPlugins []plugin.Plugin

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get working directory: %w", err)
	}
	goPath := wd // Yaegi's GoPath is set to the project's root directory.
	zap.L().Debug("Using project root as GOPATH for Yaegi.", zap.String("gopath", goPath))

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		pluginPath := filepath.Join(pluginDirPath, file.Name())
		zap.L().Debug("Attempting to load yaegi plugin.", zap.String("path", pluginPath))

		// Create a new interpreter for each plugin to isolate contexts.
		i := interp.New(interp.Options{
			GoPath: goPath,
		})

		if err := i.Use(stdlib.Symbols); err != nil {
			zap.L().Warn("Error loading stdlib symbols into yaegi.",
				zap.String("pluginPath", pluginPath),
				zap.Error(err))
			continue
		}

		if err := i.Use(symbol.Symbols); err != nil {
			zap.L().Warn("Error loading incipio symbols into yaegi.",
				zap.String("pluginPath", pluginPath),
				zap.Error(err))
			continue
		}

		srcBytes, err := os.ReadFile(pluginPath)
		if err != nil {
			zap.L().Warn("Error reading plugin file.",
				zap.String("pluginPath", pluginPath),
				zap.Error(err))
			continue
		}
		src := string(srcBytes)

		_, err = i.Eval(src)
		if err != nil {
			zap.L().Warn("Error evaluating plugin source.",
				zap.String("pluginPath", pluginPath),
				zap.Error(err))
			continue
		}

		// Assumes plugin's main package exports a 'New' function.
		v, err := i.Eval("main.New")
		if err != nil {
			zap.L().Warn("Error finding 'main.New' function in plugin.",
				zap.String("pluginPath", pluginPath),
				zap.Error(err))
			continue
		}

		newFunc, ok := v.Interface().(func() plugin.Plugin)
		if !ok {
			zap.L().Warn("Exported 'New' in plugin is not of type func() plugin.Plugin.",
				zap.String("pluginPath", pluginPath),
				zap.Any("actualType", v.Interface()))
			continue
		}

		pluginInstance := newFunc()
		if pluginInstance == nil {
			zap.L().Warn("'New' function in plugin returned nil.",
				zap.String("pluginPath", pluginPath))
			continue
		}

		if pluginInstance.Metadata().Name == "" {
			zap.L().Warn("Loaded plugin has an empty name in its metadata.",
				zap.String("pluginPath", pluginPath))
		}

		zap.L().Info("Successfully loaded yaegi plugin.",
			zap.String("name", pluginInstance.Name()),
			zap.String("keyword", pluginInstance.Keyword()),
			zap.String("path", pluginPath))
		loadedPlugins = append(loadedPlugins, pluginInstance)
	}

	return loadedPlugins, nil
}
