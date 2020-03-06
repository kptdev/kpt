# Homebrew

Update this formula to a specific release by running from the project root:

    go run ./release/formula/main.go VERSION

e.g. to point the homebrew release at the tagged release v0.10.0:

    go run ./release/formula/main.go v0.10.0

Then add commit and push `kpt.rb` to git.

