# zero-web-kit dev orchestrator (Windows)
# Usage:
#   .\tools\dev.ps1                 # backend + frontend (uses local MySQL/Redis if already running)
#   .\tools\dev.ps1 start -Docker   # optional: docker compose up MySQL/Redis first
#   .\tools\dev.ps1 start -Media    # optional: ../zms/demo_media_server
#   .\tools\dev.ps1 start -SkipBuild   # skip go build, use existing bin/zero-web-kit.exe
#   .\tools\dev.ps1 stop | status | restart

param(
    [Parameter(Position = 0)]
    [ValidateSet("start", "stop", "status", "restart", "check")]
    [string]$Action = "start",
    [switch]$Docker,
    [switch]$Media,
    [switch]$Detached,
    [switch]$NoBrowser,
    [switch]$Quiet,
    [switch]$RequireDeps,
    [switch]$SkipDepsCheck,
    [switch]$SkipBuild,
    [string]$Config = "configs\config.yaml"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent
$DevDir = Join-Path $Root ".dev"
$StateFile = Join-Path $DevDir "state.json"
$LogDir = Join-Path $DevDir "logs"
$BackendPort = 18080
$FrontendPort = 9528

function Write-Title($msg) {
    Write-Host ""
    Write-Host "== $msg ==" -ForegroundColor Cyan
}

function Ensure-DevDir {
    foreach ($d in @($DevDir, $LogDir)) {
        if (-not (Test-Path $d)) { New-Item -ItemType Directory -Path $d -Force | Out-Null }
    }
}

function Get-DevState {
    if (-not (Test-Path $StateFile)) { return $null }
    try { return Get-Content $StateFile -Raw | ConvertFrom-Json } catch { return $null }
}

function Save-DevState($state) {
    Ensure-DevDir
    $state | ConvertTo-Json | Set-Content -Path $StateFile -Encoding UTF8
}

function Test-ProcessAlive([int]$ProcessId) {
    if ($ProcessId -le 0) { return $false }
    return $null -ne (Get-Process -Id $ProcessId -ErrorAction SilentlyContinue)
}

function Get-ListenerPids([int]$port) {
    $pids = @()
    try {
        $pids = @(Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue |
            Select-Object -ExpandProperty OwningProcess -Unique)
    } catch { }
    return @($pids | Where-Object { $_ -gt 0 })
}

function Stop-PortListeners {
    param([int[]]$Ports)
    foreach ($port in $Ports) {
        foreach ($procId in (Get-ListenerPids $port)) {
            Stop-ProcessTree ([int]$procId) "port:$port" | Out-Null
        }
    }
}

function Stop-ProcessTree([int]$ProcessId, [string]$label) {
    if (-not (Test-ProcessAlive $ProcessId)) { return $false }
    & taskkill /PID $ProcessId /T /F 2>$null | Out-Null
    Write-Host "  stopped $label (pid $ProcessId)" -ForegroundColor Gray
    return $true
}

function Stop-AllDev {
    $st = Get-DevState
    if ($st) {
        if ($st.media) { Stop-ProcessTree ([int]$st.media) "media" | Out-Null }
        if ($st.frontend) { Stop-ProcessTree ([int]$st.frontend) "frontend" | Out-Null }
        if ($st.backend) { Stop-ProcessTree ([int]$st.backend) "backend" | Out-Null }
    }
    # Start-Process 记录的常是 cmd/go 包装进程；真正监听的是 node / server 子进程
    Stop-PortListeners @(18080, 9528)
    if ($st -and [int]$st.media -gt 0) {
        Stop-PortListeners @(8080)
    }
    Remove-Item $StateFile -Force -ErrorAction SilentlyContinue
}

function Test-PortsDown {
    param([int[]]$Ports)
    $up = @()
    foreach ($p in $Ports) {
        if (Test-TcpPortOpen $p) { $up += $p }
    }
    return $up
}

function Wait-TcpPort([int]$port, [int]$timeoutSec = 90) {
    $deadline = (Get-Date).AddSeconds($timeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $c = New-Object System.Net.Sockets.TcpClient
            $c.Connect("127.0.0.1", $port)
            $c.Close()
            return $true
        } catch {
            Start-Sleep -Milliseconds 800
        }
    }
    return $false
}

function Ensure-Config {
    $cfgPath = Join-Path $Root $Config
    if (Test-Path $cfgPath) { return $cfgPath }
    Write-Error "Missing $Config — create configs/config.yaml (see repo template) and optional configs/config.local.yaml for secrets"
}

function Ensure-FrontendDeps {
    if (Test-Path (Join-Path $Root "web\node_modules")) { return }
    Write-Title "npm install (first time)"
    Push-Location (Join-Path $Root "web")
    & npm install
    if ($LASTEXITCODE -ne 0) { Pop-Location; Write-Error "npm install failed" }
    Pop-Location
}

function Ensure-BackendBuilt {
    param([switch]$SkipBuild)
    $binDir = Join-Path $Root "bin"
    $exe = Join-Path $binDir "zero-web-kit.exe"
    if ($SkipBuild) {
        if (-not (Test-Path $exe)) {
            Write-Error "bin/zero-web-kit.exe not found — run without -SkipBuild"
        }
        return $exe
    }
    if (-not (Test-Path $binDir)) { New-Item -ItemType Directory -Path $binDir -Force | Out-Null }
    Write-Title "Building backend (go build)"
    Push-Location $Root
    & go build -o $exe ./cmd/server
    if ($LASTEXITCODE -ne 0) { Pop-Location; Write-Error "go build failed" }
    Pop-Location
    Write-Host "Backend built: $exe" -ForegroundColor Green
    return $exe
}

function Test-TcpPortOpen([int]$port) {
    try {
        $c = New-Object System.Net.Sockets.TcpClient
        $c.Connect("127.0.0.1", $port)
        $c.Close()
        return $true
    } catch {
        return $false
    }
}

function Test-DockerAvailable {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { return $false }
    & docker info 2>$null | Out-Null
    return $LASTEXITCODE -eq 0
}

function Show-DepsHints {
    Write-Host ""
    Write-Host "  MySQL/Redis not ready on localhost?" -ForegroundColor Yellow
    Write-Host "    A) Local install — start MySQL (:3306) + Redis (:6379), edit configs/config.yaml"
    Write-Host "    B) Have Docker —  .\tools\dev.ps1 start -Docker"
    Write-Host "    C) Check only —  .\tools\dev.ps1 check"
}

function Test-LocalDeps {
    param([switch]$Quiet)
    $mysql = Test-TcpPortOpen 3306
    $redis = Test-TcpPortOpen 6379
    if (-not $Quiet) {
        Write-Title "Dependencies (MySQL / Redis)"
        if ($mysql) { Write-Host "  MySQL :3306  OK" -ForegroundColor Green }
        else { Write-Host "  MySQL :3306  not reachable" -ForegroundColor Yellow }
        if ($redis) { Write-Host "  Redis :6379  OK" -ForegroundColor Green }
        else { Write-Host "  Redis :6379  not reachable" -ForegroundColor Yellow }
    }
    return @{ MySql = $mysql; Redis = $redis; Ok = ($mysql -and $redis) }
}

function Ensure-LocalDeps {
    if ($SkipDepsCheck) { return }
    $deps = Test-LocalDeps
    if ($deps.Ok) { return }
    Show-DepsHints
    if ($RequireDeps) {
        Write-Error "MySQL/Redis required (-RequireDeps). Fix deps or use -Docker if available."
    }
    Write-Host ""
    Write-Host "  Continuing without deps (backend may fail until DB is up)..." -ForegroundColor Gray
    if (-not $Detached) {
        $ans = Read-Host "Continue? [y/N]"
        if ($ans -notmatch '^[yY]') { exit 0 }
    }
}

function Start-DockerDeps {
    if (-not (Test-DockerAvailable)) {
        Write-Error @"
Docker is not available (not installed or daemon not running).
  - Use local MySQL/Redis:  .\tools\dev.ps1 start
  - Or install/start Docker Desktop, then:  .\tools\dev.ps1 start -Docker
"@
    }
    Write-Title "Docker: MySQL + Redis"
    Push-Location (Join-Path $Root "docker")
    & docker compose up -d
    if ($LASTEXITCODE -ne 0) { Pop-Location; Write-Error "docker compose up failed" }
    Pop-Location
    Write-Host "Waiting for MySQL :3306 ..."
    if (Wait-TcpPort 3306 120) {
        Write-Host "MySQL ready" -ForegroundColor Green
    } else {
        Write-Warning "MySQL :3306 not ready yet — check docker logs"
    }
    if (Wait-TcpPort 6379 30) {
        Write-Host "Redis ready" -ForegroundColor Green
    } else {
        Write-Warning "Redis :6379 not ready yet"
    }
}

function Find-MediaServer {
    $zmsRoot = Join-Path (Split-Path $Root -Parent) "zms"
    foreach ($rel in @(
            "build\examples\Release\demo_media_server.exe",
            "build\examples\Debug\demo_media_server.exe",
            "build\examples\demo_media_server.exe"
        )) {
        $p = Join-Path $zmsRoot $rel
        if (Test-Path -LiteralPath $p) { return @{ Exe = $p; Root = $zmsRoot } }
    }
    return $null
}

function Start-BackgroundLogged {
    param(
        [string]$Tag,
        [string]$FileName,
        [string[]]$ArgumentList,
        [string]$WorkingDirectory
    )
    $outLog = Join-Path $LogDir "$Tag.out.log"
    $errLog = Join-Path $LogDir "$Tag.err.log"
    foreach ($f in @($outLog, $errLog)) {
        if (Test-Path $f) { Remove-Item $f -Force }
    }
    return Start-Process -FilePath $FileName -ArgumentList $ArgumentList `
        -WorkingDirectory $WorkingDirectory `
        -RedirectStandardOutput $outLog -RedirectStandardError $errLog `
        -PassThru -WindowStyle Hidden
}

function Init-LogOffsetsToEnd {
    $script:LogOffsets = @{}
    Get-ChildItem $LogDir -Filter "*.log" -ErrorAction SilentlyContinue | ForEach-Object {
        $script:LogOffsets[$_.FullName] = [int64]$_.Length
    }
}

function Test-LogLineNoise([string]$line) {
    if ($line -match '\[webpack\.Progress\]') { return $true }
    if ($line -match '^\s*INFO\s+Starting development server\.\.\.') { return $true }
    if ($line -match 'To create a production build, run npm run build') { return $true }
    if ($line -match 'DeprecationWarning.*util\._extend') { return $true }
    return $false
}

function Show-NewLogLines {
    Get-ChildItem $LogDir -Filter "*.log" -ErrorAction SilentlyContinue | ForEach-Object {
        $path = $_.FullName
        $tag = ($_.BaseName -replace '\.(out|err)$', '')
        if (-not $script:LogOffsets.ContainsKey($path)) {
            $script:LogOffsets[$path] = 0L
        }
        if ($_.Length -le $script:LogOffsets[$path]) { return }

        $fs = [System.IO.File]::Open($path, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::ReadWrite)
        try {
            $null = $fs.Seek($script:LogOffsets[$path], [IO.SeekOrigin]::Begin)
            $reader = New-Object System.IO.StreamReader($fs, [Text.Encoding]::UTF8, $true)
            while ($null -ne ($line = $reader.ReadLine())) {
                if ($line -match '\S' -and -not (Test-LogLineNoise $line)) {
                    Write-Host "[$tag] $line" -ForegroundColor DarkGray
                }
            }
            $script:LogOffsets[$path] = $fs.Position
        } finally {
            $fs.Close()
        }
    }
}

function Do-Start {
    $existing = Get-DevState
    if ($existing -and ((Test-ProcessAlive ([int]$existing.backend)) -or (Test-ProcessAlive ([int]$existing.frontend)))) {
        Write-Host "Already running. Use: .\tools\dev.ps1 status | stop" -ForegroundColor Yellow
        return
    }
    Stop-AllDev
    Ensure-DevDir
    Ensure-Config | Out-Null
    Ensure-FrontendDeps
    if ($Docker) {
        Start-DockerDeps
    } else {
        Ensure-LocalDeps
    }

    $cfgRel = ($Config -replace '\\', '/')
    $backendExe = Ensure-BackendBuilt -SkipBuild:$SkipBuild

    Write-Title "Starting backend :$BackendPort"
    $backend = Start-BackgroundLogged -Tag "backend" -FileName $backendExe `
        -ArgumentList @("-config", $cfgRel) -WorkingDirectory $Root

    if (-not (Wait-TcpPort $BackendPort 60)) {
        Get-Content (Join-Path $LogDir "backend.err.log") -ErrorAction SilentlyContinue
        Stop-ProcessTree $backend.Id "backend"
        Write-Error "Backend failed on :$BackendPort — see .dev/logs/backend.err.log"
    }
    Write-Host "Backend ready (pid $($backend.Id))" -ForegroundColor Green

    Write-Title "Starting frontend :$FrontendPort"
    # BROWSER=none: vue.config.js has open:true; we open once from this script
    $frontend = Start-BackgroundLogged -Tag "frontend" -FileName "cmd.exe" `
        -ArgumentList @("/c", "set BROWSER=none&& npm run dev") `
        -WorkingDirectory (Join-Path $Root "web")

    if (-not (Wait-TcpPort $FrontendPort 120)) {
        Get-Content (Join-Path $LogDir "frontend.err.log") -ErrorAction SilentlyContinue
        Stop-ProcessTree $frontend.Id "frontend"
        Stop-ProcessTree $backend.Id "backend"
        Write-Error "Frontend failed on :$FrontendPort — see .dev/logs/frontend.err.log"
    }
    Write-Host "Frontend ready (pid $($frontend.Id))" -ForegroundColor Green

    $mediaPid = 0
    if ($Media) {
        $zms = Find-MediaServer
        if (-not $zms) {
            Write-Warning "demo_media_server not found under ../zms/build — skip -Media"
        } else {
            $ini = Join-Path $zms.Root "conf\config.zero-web-kit.ini"
            $cfgArg = if (Test-Path $ini) { @("--config", "conf/config.zero-web-kit.ini") } else { @("--config", "conf/config.ini") }
            Write-Title "Starting zero-media-server :8080"
            $media = Start-BackgroundLogged -Tag "media" -FileName $zms.Exe `
                -ArgumentList $cfgArg -WorkingDirectory $zms.Root
            $mediaPid = $media.Id
            Write-Host "Media server (pid $mediaPid)" -ForegroundColor Green
        }
    }

    Save-DevState @{
        backend  = if (($bk = (Get-ListenerPids $BackendPort | Select-Object -First 1))) { $bk } else { $backend.Id }
        frontend = if (($fe = (Get-ListenerPids $FrontendPort | Select-Object -First 1))) { $fe } else { $frontend.Id }
        media    = $mediaPid
        started  = (Get-Date).ToString("o")
    }

    Write-Title "Dev stack running"
    Write-Host "  UI:    http://localhost:$FrontendPort"
    Write-Host "  API:   http://localhost:$BackendPort"
    Write-Host "  Logs:  $LogDir"
    Write-Host "  Stop:  .\tools\dev.ps1 stop"
    if (-not $NoBrowser) { Start-Process "http://localhost:$FrontendPort" }

    if ($Detached) {
        Write-Host "Detached — processes run in background." -ForegroundColor Gray
        return
    }

    Write-Host ""
    if ($Quiet) {
        Write-Host "Quiet mode — logs in $LogDir (Ctrl+C stops all)" -ForegroundColor Gray
    } else {
        Write-Host "Streaming new log lines only (Ctrl+C stops all). Use -Quiet for a silent watch." -ForegroundColor Gray
        Init-LogOffsetsToEnd
    }
    try {
        while ($true) {
            if (-not (Test-ProcessAlive $backend.Id)) {
                Write-Warning "Backend exited"
                break
            }
            if (-not (Test-ProcessAlive $frontend.Id)) {
                Write-Warning "Frontend exited"
                break
            }
            if (-not $Quiet) { Show-NewLogLines }
            Start-Sleep -Seconds 1
        }
    } finally {
        Stop-AllDev
    }
}

function Do-Status {
    $st = Get-DevState
    Write-Title "zero-web-kit dev status"
    if (-not $st) {
        Write-Host "  Not started via dev.ps1"
    } else {
        foreach ($row in @(
                @{ name = "backend"; pid = [int]$st.backend; port = $BackendPort },
                @{ name = "frontend"; pid = [int]$st.frontend; port = $FrontendPort },
                @{ name = "media"; pid = [int]$st.media; port = 8080 }
            )) {
            if ($row.pid -le 0) { continue }
            $alive = Test-ProcessAlive $row.pid
            $portOk = Wait-TcpPort $row.port 2
            $s = if ($alive) { "running" } else { "stopped" }
            Write-Host ("  {0,-10} pid={1,-6} {2,-8} :{3} open={4}" -f $row.name, $row.pid, $s, $row.port, $portOk) `
                -ForegroundColor $(if ($alive) { "Green" } else { "Red" })
        }
    }
    Write-Host ""
    Write-Host "  http://localhost:$FrontendPort"
}

function Do-Check {
    Test-LocalDeps | Out-Null
    Write-Host ""
    if (Test-DockerAvailable) {
        Write-Host "  Docker: available (use -Docker to start MySQL/Redis containers)" -ForegroundColor Green
    } else {
        Write-Host "  Docker: not available — use local MySQL/Redis, or install Docker Desktop" -ForegroundColor Yellow
    }
    $deps = @{ MySql = (Test-TcpPortOpen 3306); Redis = (Test-TcpPortOpen 6379) }
    if (-not ($deps.MySql -and $deps.Redis)) { Show-DepsHints }
}

Set-Location $Root
switch ($Action) {
    "start" { Do-Start }
    "stop" {
        Write-Title "Stopping zero-web-kit (API :18080, UI :9528)"
        Stop-AllDev
        $still = Test-PortsDown @(18080, 9528)
        if ($still.Count -eq 0) {
            Write-Host "Done." -ForegroundColor Green
        } else {
            Write-Warning "Ports still open: $($still -join ', ') — run as Admin or kill manually"
        }
        if (Test-TcpPortOpen 8080) {
            Write-Host "  :8080 still up (zero-media-server) — video may work without web-kit; stop ZMS separately" -ForegroundColor Yellow
        }
    }
    "status" { Do-Status }
    "restart" { Stop-AllDev; Do-Start }
    "check" { Do-Check }
}
