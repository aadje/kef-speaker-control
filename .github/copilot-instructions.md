# KEF Speaker Control Project Instructions

## Project Overview

This is a **dual-language CLI tool** for controlling KEF LS50 Wireless II speakers over HTTP. The primary implementation is in Go ([main.go](../main.go)), with a legacy PowerShell version in `powershell/`. The Go version was converted from PowerShell by a coding agent to provide better cross-platform support.

## Architecture

**Single-file Go application** with no external dependencies. All logic resides in [main.go](../main.go):
- HTTP client wrapper functions (`getData`, `setData`, `setDataInt`)
- Command parser with aliases (`start`/`on` → `coaxial`, `stop`/`off` → `standby`)
- KEF API response handling (handles both array and single-object responses)

The KEF API is quirky:
- All operations use HTTP GET (even for mutations)
- `/api/setData` endpoint with URL-encoded JSON in query params
- Inconsistent response formats (sometimes array, sometimes single object)
- Boolean values are strings: `"True"` / `"False"`

## Critical Configuration

**KEF_URL environment variable** is the primary config mechanism:
```bash
export KEF_URL=http://192.168.0.135
```
Default fallback: `http://192.168.0.105`

## Key Commands & Workflows

### Building
```bash
go build                           # Creates 'kef' binary
sudo mv kef /usr/local/bin/        # Optional: install to PATH
```

### Testing API Behavior
The speaker API requires actual hardware. Test with:
```bash
kef show                           # Status check
kef scan                           # Network discovery
```

## Code Patterns

### KEF API Request Pattern
All API calls follow this structure:
```go
// GET with URL-encoded JSON value parameter
/api/setData?path=<path>&roles=value&value={"type":"<type>","<type>":"<value>"}
```

Example from [main.go](../main.go):
```go
func setData(baseURL, path, dataType, value string) (*SetDataResponse, error) {
    jsonValue := fmt.Sprintf(`{"type":"%s","%s":"%s"}`, dataType, dataType, value)
    urlStr := fmt.Sprintf("%s/api/setData?path=%s&roles=value&value=%s",
        baseURL, url.QueryEscape(path), url.QueryEscape(jsonValue))
    // ...
}
```

### Response Handling Pattern
KEF API returns inconsistent formats. Always handle both:
```go
// Try array first
var arrayResult []GetDataResponse
if err := json.Unmarshal(body, &arrayResult); err == nil && len(arrayResult) > 0 {
    return arrayResult, nil
}
// Fall back to single object
var singleResult GetDataResponse
// ...
```

### Command Alias System
Commands have multiple aliases for better UX:
- `on`/`start` maps to default source (typically `coaxial`)
- `off`/`stop` maps to `standby`
- Numeric strings (0-100) set volume

## KEF API Endpoints

Reference from [main.go](../main.go):
- `settings:/deviceName` - device name
- `settings:/kef/play/physicalSource` - source selection (coaxial, wifi, bluetooth, tv, optic, analog, standby)
- `player:volume` - volume level (0-100)
- `settings:/mediaPlayer/mute` - mute state (bool string)

## Network Scanning

The `scan` command implements basic /24 subnet scanning by:
1. Detecting default gateway (tries common addresses)
2. Testing all 254 IPs in subnet
3. Checking for HTTP server on port 80
4. Validating response contains "LS50 Wireless II"

Not production-grade, but works for local discovery. See [main.go](../main.go) `scanNetwork()`.

## Platform Differences

Go version is cross-platform. PowerShell version (`powershell/kef.psm1`) requires PowerShell Core 7+ and uses `Invoke-RestMethod` instead of `net/http`.

## When Making Changes

- Maintain single-file simplicity (no packages/modules unless necessary)
- Test against actual hardware (API has no mock/sandbox mode)
- Preserve command alias system for backward compatibility
- Handle both API response formats (array/object)
- Keep `KEF_URL` as primary config method
