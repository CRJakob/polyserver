# PolyServer

## Usage
To run the server:
`go run .`

args:
`-port <port>`: the port for the web dashboard frontend. Default is 8080
`-control-port <port>`: the port for the internal API. Default is 9090
`-tracks <path/to/dir>` the directory containing .track files for the server to load

## Debugging
To log to a file, simply run the server and redirect output to a file.
### Windows
PowerShell: `go run . *> polyserver.log`
Then to view the log live: `Get-Content polyserver.log -Wait`

### Linux/MacOS
`go run . > polyserver.log 2>&1`
To view live: `tail -f polyserver.log`

## To quit the server and close the dashboard, just use your OS's kill keybind (By default Ctrl + C on Windows, Linux and MacOS) in the Terminal tab