# autoclicker

go build -ldflags="-s -w -H=windowsgui -X main.version=0.1.5" -trimpath .

repomix --remove-comments --remove-empty-lines --include "**/*.go"
