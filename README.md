# KEF LS50 Wireless II Controller

A Go application to control KEF LS50 Wireless II speakers on your local network.

## Features

- Control speaker source (coaxial, wifi, bluetooth, TV, optical, analog)
- Adjust volume (0-100)
- Mute/unmute
- Display current status
- Scan local network for KEF speakers
- Turn on/off (standby mode)

## Installation
From source with GO installed
```bash
# Build the application
go build

# Optional: Install to your PATH
sudo mv kef /usr/local/bin/
```

## Configuration

Set the `KEF_URL` environment variable to your speaker's IP address.  
If you don't know your speaker's IP address, you can try the scan command.

```bash
kef scan
```
This will scan your local subnet and find KEF speakers. When found, it will display the URL you should set as `KEF_URL`.  
It scans a /24 subnet based on your local network ip. By simply generating 256 ip's and replacing the number after the last dot and test if a Kef speaker responds to an http request. 
This is not very advanced but worked on My Machines. You could also try finding the speakers in your File Explorer or install the NMAP cli network scanner to find your speaker IP. 

Configure your speakers address in an Environment variable
```bash
export KEF_URL=http://192.168.0.135
```

To make it permanent between shell's, add the env var to your `~/.bashrc` or `~/.zshrc` file
```bash
echo 'export KEF_URL=http://192.168.0.135' >> ~/.bashrc
```

## Usage

### Basic Commands

```bash
# Show current status (default)
kef
kef show

# Turn on (sets to coaxial source)
kef start
kef on

# Turn off (standby mode)
kef stop
kef off
kef standby

# Change source
kef coaxial
kef wifi
kef bluetooth
kef tv
kef optic
kef analog

# Set volume (0-100)
kef 0
kef 40
kef 100

# Adjust volume
kef up    # Increase by 10
kef down  # Decrease by 10

# Mute control
kef mute
kef unmute

# Open speaker web interface
kef browse

# Scan local network for KEF speakers
kef scan
```

## Examples

```bash
# Check speaker status
$ kef show
Device: LS50 Wireless II
Source: coaxial
Volume: 40
Muted : false

# Set volume to 50
$ kef 50
Volume: 50

# Switch to Bluetooth
$ kef bluetooth
Source : bluetooth

# Mute the speaker
$ kef mute
Muted : true
```

## Notes

- The KEF LS50 Wireless II API is quite basic and uses HTTP GET requests to control the speaker
- No authentication is required
- All commands use the `/api/setData` and `/api/getData` endpoints
- The API response format varies (sometimes arrays, sometimes objects), which is handled in the code

## Links

- [Original PowerShell version](/powershell/)
- [Roon Labs Discussion](https://community.roonlabs.com/t/ls50-wireless-ii-home-automation/154388)
- [Python Client](https://github.com/N0ciple/pykefcontrol)
