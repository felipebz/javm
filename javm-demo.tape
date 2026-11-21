Output assets/demo.gif

Set FontSize 20
Set Width 1200
Set Height 400
Set Padding 20
Set FontFamily "JetBrainsMono Nerd Font"
Set Theme "Catppuccin Mocha"
Set CursorBlink false

# Setup: empty javm data directory and a project that asks for Java 25
Hide
Type 'rm -rf ~/.cache/javm-demo && export JAVM_HOME=~/.cache/javm-demo/home && mkdir -p ~/.cache/javm-demo/project && cd ~/.cache/javm-demo/project && echo 25 > .java-version'
Enter
Ctrl+L
Show

# Run
Type "cat .java-version"
Enter
Sleep 1s

Type "javm install"
Enter
Wait@5m

Type "javm exec java -version"
Enter
Sleep 4s
