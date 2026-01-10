Output assets/demo.gif

Set FontSize 20
Set Width 1200
Set Height 450
Set Padding 20
Set FontFamily "JetBrains Mono"
Set Theme "Catppuccin Mocha"
Set CursorBlink false

# Setup
Hide
Type 'eval "$(javm init bash)"'
Enter
Hide
Type 'javm use 21'
Enter
Ctrl+L
Show

# Run
Type "java -version"
Enter
Sleep 300ms

Type "javm install 25"
Enter
Sleep 7s

Type "javm use 25"
Enter
Sleep 300ms

Type "java -version"
Enter
Sleep 3s
