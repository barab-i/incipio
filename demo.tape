Output ./assets/demo.gif

Set FontSize 18
Set Width 1200
Set Height 600
Set TypingSpeed 200ms

# Start incipio with plugins
Type "incipio --plugins=wikipedia,nixshell"
Enter
Sleep 1.5s # Wait for startup

# Navigate initial options/suggestions
Down
Sleep 200ms
Down
Sleep 2s # Pause after navigation

# Perform a calculation
Type "= (1+1)^4"
Sleep 3s # Show calculation result

# Exit calculation mode
Escape
Sleep 500ms

# Search Wikipedia
Type "!w life could"
Sleep 1s

# Confirm search
Enter
Sleep 5s # Show Wikipedia result

# Exit Wikipedia view
Escape
Sleep 500ms

# Use Nix shell plugin
Type "!n telegram-desktop"
Sleep 4s # Show command/result