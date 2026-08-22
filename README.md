# autoclicker

go build -ldflags="-s -w -H=windowsgui -X main.version=0.1.6" -trimpath .

repomix --remove-comments --remove-empty-lines --include "**/*.go"
