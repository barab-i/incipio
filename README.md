# Incipio

Incipio is a command-line launcher application built with Go, featuring a modular architecture and a terminal user interface powered by Bubble Tea.

![](./assets/demo.gif)

## Features

*   **Modular Design:** The application is structured with distinct plugins for different functionalities.
*   **Terminal User Interface:** Interactive TUI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and styled with [Lipgloss](https://github.com/charmbracelet/lipgloss).
*   **Plugins:** Comes with several useful plugins out-of-the-box:
    *   **App Launcher:** Finds and launches desktop applications.
    *   **Calculator:** Performs basic arithmetic calculations.
    *   **Plugin Manager:** Allows enabling/disabling optional plugins.
    *   **Wikipedia Search:** Searches Wikipedia for articles (example plugin, located in `examples/plugins/`).
    *   **Nix Shell:** Provides an interface to launch applications with `nix shell` (example plugin, located in `examples/plugins/`).

> [!WARNING]
> **Nix Shell Plugin:** Requires the `nix-locate` command (part of the `nix-index` package) to be installed and available in your PATH. Generating the `nix-locate` database using `nix-index` is also required for it to function correctly.

## Usage

### Configuring Sway Keybinding

To launch Incipio using a keyboard shortcut in the Sway window manager, add the following line to your Sway configuration file:

```sh
# ~/.config/sway/config

# Set your terminal (e.g., wezterm, kitty)
set $term wezterm start

# Optional: Make the launcher window floating
for_window [app_id="^incipio$"] floating enable, sticky enable, resize set 50 ppt 60 ppt

# Define the command to launch Incipio within the terminal
#    Use --class or --app-id depending on your terminal and preference for matching the window rule.
#    Use the --plugins flag to specify which plugins to load (comma-separated).
set $menu $term --class incipio-launcher -e incipio
# Example using foot terminal with app-id:
# set $menu foot --app-id incipio-launcher -e incipio

bindsym $mod+d exec $menu

```

## Plugins

Incipio features a flexible plugin system that allows for extending its functionality. Plugins can be either built-in or loaded dynamically at runtime using [Yaegi](https://github.com/traefik/yaegi).

### Plugin Types

*   **Built-in Plugins:** These are compiled directly into the Incipio binary and are always available. Core functionalities like the App Launcher ([`internal/plugins/applauncher/launcher.go`](internal/plugins/applauncher/launcher.go)), Calculator ([`internal/plugins/calculator/calculator.go`](internal/plugins/calculator/calculator.go)), and the Plugin Manager itself ([`internal/plugins/pluginmanager/pluginmanager.go`](internal/plugins/pluginmanager/pluginmanager.go)) are implemented as built-in plugins.
*   **Yaegi Plugins:** These are external Go files (`.go`) that are interpreted at runtime. This allows users to add custom functionality without recompiling Incipio.
    *   Place your custom Yaegi plugins in `~/.config/incipio/plugins/`.
    *   Example Yaegi plugins can be found in the [`examples/plugins/`](examples/plugins/) directory of this repository (e.g., [`hello.go`](examples/plugins/hello.go), [`wikipedia.go`](examples/plugins/wikipedia.go), [`nixshell.go`](examples/plugins/nixshell.go)). These serve as templates for creating your own.

### Enabling Optional Plugins

Some plugins are optional and can be enabled at startup using the `--plugins` command-line flag. Provide a comma-separated list of plugin flags. For example:

```sh
incipio --plugins=wikipedia,nixshell
```

## Building

To build Incipio from source, you need Go installed (version 1.24.2).

```sh
git clone https://github.com/barab-i/incipio
cd incipio
go build ./cmd/incipio
```

### 1. Using `nix profile install` (User Environment)

This installs the application only for the current user.

```sh
# Install from the local flake in the current directory
nix profile install .#incipio

# Or, if you have pushed the flake to a remote repository (e.g., GitHub)
nix profile install github:barab-i/incipio
```

### 2. Adding to NixOS Configuration (System-wide)

If you are using NixOS, you can add Incipio to your system configuration (`/etc/nixos/configuration.nix` or wherever your configuration resides).

First, add the flake as an input in your main `flake.nix`:

```nix
// filepath: /path/to/your/flake.nix
{
  description = "My NixOS Configuration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    # Add incipio flake input
    incipio = {
      url = "github:barab-i/incipio";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    # ... other inputs like home-manager etc.
  };

  outputs = { self, nixpkgs, incipio, ... }@inputs: {
    nixosConfigurations.your-hostname = nixpkgs.lib.nixosSystem {
      # ... system configuration ...
      specialArgs = { inherit inputs; }; # Pass inputs to modules
      modules = [
        ./configuration.nix
        # ... other modules
      ];
    };
  };
}
```

Then, add the package to your environment.systemPackages in configuration.nix:

```nix
// filepath: /path/to/your/configuration.nix
{ config, pkgs, inputs, ... }:

{
  # ... other configuration options ...

  environment.systemPackages = [
    # ... other packages ...
    inputs.incipio.packages.${pkgs.system}.default
  ];

  # ... rest of configuration ...
}
```

### 3. Adding to Home Manager Configuration
If you use Home Manager, either standalone or as a NixOS module, you can add Incipio to your user environment.

Add the flake input similarly to the NixOS example above (either in your system `flake.nix` if using the module, or your standalone home-manager `flake.nix`).

Then, add the package to `home.packages` in your `home.nix`:

```nix
// filepath: /path/to/your/home.nix
{ config, pkgs, inputs, ... }:

{
  # ... other home-manager options ...

  home.packages = [
    # ... other packages ...
    inputs.incipio.packages.${pkgs.system}.default
  ];

  # ... rest of configuration ...
}
```

## Theming
Incipio allows customization of its appearance through theme files based on the [Base16 Styling Guidelines](https://github.com/chriskempson/base16/blob/main/styling.md).

The application looks for a theme.yaml file in the XDG config directory (`~/.config/incipio/theme.yaml by default`). You can place a Base16 theme definition in this file to change the application's colors.

A wide variety of pre-built Base16 themes can be found at [tinted-theming/base16-schemes](https://github.com/tinted-theming/base16-schemes).

## Roadmap

### Done
*   [x] Modular plugin architecture ([`internal/app/plugin_manager.go`](internal/app/plugin_manager.go), [`pkgs/plugin/interface.go`](pkgs/plugin/interface.go))
*   [x] Terminal User Interface with Bubble Tea ([`internal/app/model.go`](internal/app/model.go))
*   [x] Built-in plugins:
    *   [x] App Launcher ([`internal/plugins/applauncher/launcher.go`](internal/plugins/applauncher/launcher.go))
    *   [x] Calculator ([`internal/plugins/calculator/calculator.go`](internal/plugins/calculator/calculator.go))
    *   [x] Plugin Manager (view status) ([`internal/plugins/pluginmanager/pluginmanager.go`](internal/plugins/pluginmanager/pluginmanager.go))
*   [x] Dynamic plugin loading with Yaegi ([`internal/yaegi/yaegi.go`](internal/yaegi/yaegi.go))
*   [x] Base16 Theming support ([`internal/theme/theme.go`](internal/theme/theme.go))
*   [x] Command-line flag for enabling optional plugins ([`cmd/incipio/main.go`](cmd/incipio/main.go))

### To Do
*   [ ] Detailed documentation for creating custom Yaegi plugins.
*   [ ] More built-in plugins (e.g., File Browser, Clipboard Manager).
*   [ ] Persistent configuration for plugins (beyond CLI flags).
*   [ ] UI for enabling/disabling/configuring plugins directly within the Plugin Manager plugin.
*   [ ] **Plugin Manager Plugin**: Add option to discover and install Yaegi plugins from a curated online repository or user-defined URLs.
*   [ ] Asynchronous plugin loading to improve startup time.
*   [ ] More sophisticated layout options for plugins that render their own views.
