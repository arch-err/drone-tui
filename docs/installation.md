# Installation

## Binary Releases

Download the latest binary from the [releases page](https://github.com/arch-err/drone-tui/releases).

=== "Linux (amd64)"

    ```bash
    curl -LO https://github.com/arch-err/drone-tui/releases/latest/download/drone-tui_linux_amd64.tar.gz
    tar xzf drone-tui_linux_amd64.tar.gz
    sudo mv drone-tui /usr/local/bin/
    ```

=== "macOS (Apple Silicon)"

    ```bash
    curl -LO https://github.com/arch-err/drone-tui/releases/latest/download/drone-tui_darwin_arm64.tar.gz
    tar xzf drone-tui_darwin_arm64.tar.gz
    sudo mv drone-tui /usr/local/bin/
    ```

## From Source

Requires Go 1.21+:

```bash
go install github.com/arch-err/drone-tui/cmd/drone-tui@latest
```
