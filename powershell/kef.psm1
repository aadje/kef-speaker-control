#Requires -PSEdition Core
#Requires -Version 7

function kef {
    <#
    .SYNOPSIS
        Control a KEF LS50 Wireless II speaker in your local network
    .DESCRIPTION
        Import this Module or copy the kef Function into your Powershell Core profile

        nmap -v -sn 192.168.0.0/24

        ipmo .\kef.psm1 -Force
        code $PROFILE.CurrentUserAllHosts
    .NOTES
        Ensure the KefBaseUrl is set. This url can be found using `kef scan` which scans your local network. 
        This script uses the poorly designed api of the LS50 II. All commands use http GET's to a single `/api/setData` endpoint.
        No authentication required. My Speakers are running at http://192.168.0.135 
    .LINK
        https://gist.github.com/aadje/15945bc5a55035bc6a4f22270241b3f8

        https://community.roonlabs.com/t/ls50-wireless-ii-home-automation/154388
        https://github.com/pirminj/ls50w2_client
        https://github.com/N0ciple/pykefcontrol
    .EXAMPLE
        kef
        kef start
        kef bluetooth
        kef 40
        kef up
        kef mute
        kef browse
        kef scan
    #>
    param (
        [ValidateSet(
            'on','off','start','stop','standby','coaxial','wifi','bluetooth','tv','optic','analog',
            '0','10','20','30','40','50','60','70','80','90','100',
            'show','up','down','mute','unmute','browse','scan','nmap'
        )]
        [string] $Command = 'show',
        [string] $KefBaseUrl = $env:KEF_URL ?? 'http://192.168.0.105',
        [string] $StartSource = 'coaxial'
    )

    if($Command -in 'start', 'on') { $Command = $StartSource }
    if($Command -in 'stop', 'off') { $Command = 'standby' }

    function GetData ($Path) { 
        Invoke-RestMethod "$KefBaseUrl/api/getData?path=$Path&roles=value" 
    }
    function SetData ($Path, $Type, $value) { 
        Invoke-RestMethod "$KefBaseUrl/api/setData?path=$Path&roles=value&value={`"type`":`"$Type`",`"$Type`":`"$value`"}" 
    }

    switch ($Command) {
        'show' {
            Write-Host "Device: $((GetData 'settings:/deviceName').string_)"
            Write-Host "Source: $((GetData 'settings:/kef/play/physicalSource').kefPhysicalSource)"
            Write-Host "Volume: $((GetData 'player:volume')[0].i32_)"
            Write-Host "Muted : $((GetData 'settings:/mediaPlayer/mute')[0].bool_)"
        }

        ({$Command -in 'standby', 'coaxial', 'wifi', 'bluetooth', 'tv', 'optic', 'analog'}) {
            $result = SetData 'settings:/kef/play/physicalSource' 'kefPhysicalSource' $Command
            Write-Host "Source : $($result.value.kefPhysicalSource)"
        }

        ({$Command -as [int]}) {
            $volume = [int]$Command
            if($volume -ge 0 -and $volume -le 100){
                $success = SetData 'player:volume' 'i32_' $volume
                if($success) {Write-Host "Volume: $volume"}
            } else {
                Write-Host "Volume '$volume' unknown, use value between 1 and 100"
            }
        }

        ({$Command -in 'up', 'down'}) {
            $volume = (GetData 'player:volume')[0].i32_
            if($Command -eq 'up' -and $volume -le 90){
                $volume+=10
            }
            if($Command -eq 'down' -and $volume -ge 10){
                $volume-=10
            }
            $success = SetData 'player:volume' 'i32_' $volume
            if($success) {Write-Host "Volume: $volume"}
        }

        ({$Command -in 'mute', 'unmute'}) {
            if($Command -eq 'mute')     { $Value = 'True' }
            if($Command -eq 'unmute')   { $Value = 'False' }
            $result = SetData 'settings:/mediaPlayer/mute' 'bool_' $Value
            Write-Host "Muted : $($result.value.bool_)"
        }

        'browse' {
            Start-Process $KefBaseUrl
        }

        'scan' {
            $gatewayAddress = Get-NetIPConfiguration | ForEach-Object IPv4DefaultGateway | Select-Object -Unique -ExpandProperty NextHop -First 1
            $subRange = $gatewayAddress.SubString(0, $gatewayAddress.LastIndexOf('.'))
            $range = 0..254
            $range | ForEach-Object { 
                $address = "$subRange.$_"
                $webAddress = "http://$address"
                Write-Progress "Scanning $subRange.0/24 network subnet" $webAddress -PercentComplete ((($_-$range[0])/$range.Count)*100)
                $tcpClient = [System.Net.Sockets.TcpClient]::new()
                $tcpClient.ConnectAsync($address, 80).Wait([Timespan]::FromMilliseconds(100)) | Out-Null
                if($tcpClient.Connected){
                    $response = Invoke-WebRequest $webAddress -SkipHttpErrorCheck -ErrorAction SilentlyContinue
                    if($response.Content -match "LS50 Wireless II"){
                        $env:KEF_URL = $webAddress
                        Write-Host "Found KEF speakers at $webAddress"
                        break
                    }
                }
            }
            Write-Host "KEF speakers not found in $subRange.0/24 network range"
        }
        'nmap' {
            nmap -sn 192.168.0.125/24
        }
    }
}
