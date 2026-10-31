# Compiles winres/asusimemode.manifest into rsrc_windows_amd64.syso and rsrc_windows_arm64.syso in
# the repository root, where `go build`, `go install` and GoReleaser link them in automatically
# (Go's linker has no manifest option of its own). Run by `task winres` after the manifest
# changes; the .syso files are committed, so building needs none of this.
#
# Uses only Microsoft's own tools, from the Visual Studio Build Tools C++ workload: rc.exe
# (Windows SDK) compiles the .rc into a .res, and cvtres.exe (MSVC) converts it into a COFF
# object per architecture. Two parts of that object vary from run to run and machine to machine,
# so both are zeroed and the same manifest always gives the same bytes:
#   - the time stamp in the COFF header;
#   - the .debug$S section, CodeView records holding absolute paths (the temporary .res, the
#     output file, the working directory, cvtres.exe). Go's linker only takes the .rsrc$01 and
#     .rsrc$02 sections, and the paths would otherwise end up in the committed files.
# And one symbol Go's linker refuses: @comp.id, the tool-version stamp, is an absolute symbol
# (section -1), which fails the link with "sectnum < 0!". It is marked as a debug symbol
# (section -2), which the linker skips (cmd/link/internal/loadpe, as it does @feat.00).
#
# Keep this file ASCII: Windows PowerShell 5.1 reads BOM-less files as ANSI.
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
$vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'
$vs = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
if (-not $vs) { throw 'Visual Studio Build Tools with the C++ workload not found' }
$cvtres = Get-ChildItem (Join-Path $vs 'VC\Tools\MSVC\*\bin\Hostx64\x64\cvtres.exe') |
    Sort-Object FullName | Select-Object -Last 1
$rc = Get-ChildItem (Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\bin\10.*\x64\rc.exe') |
    Sort-Object FullName | Select-Object -Last 1
if (-not $cvtres -or -not $rc) { throw 'rc.exe or cvtres.exe not found' }

$tmp = Join-Path ([IO.Path]::GetTempPath()) "asusimemode-winres-$PID"
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
    $res = Join-Path $tmp 'asusimemode.res'
    & $rc.FullName /nologo /fo $res (Join-Path $PSScriptRoot 'asusimemode.rc')
    if ($LASTEXITCODE) { throw "rc.exe failed ($LASTEXITCODE)" }

    foreach ($arch in @{ amd64 = 'X64'; arm64 = 'ARM64' }.GetEnumerator()) {
        $out = Join-Path $root "rsrc_windows_$($arch.Key).syso"
        & $cvtres.FullName /nologo "/machine:$($arch.Value)" "/out:$out" $res
        if ($LASTEXITCODE) { throw "cvtres.exe failed for $($arch.Key) ($LASTEXITCODE)" }
        $bytes = [IO.File]::ReadAllBytes($out)
        # IMAGE_FILE_HEADER: TimeDateStamp at 4, NumberOfSections at 2, and the 40-byte section
        # headers follow the 20-byte header (an object file has no optional header)
        [Array]::Clear($bytes, 4, 4)
        $sections = [BitConverter]::ToUInt16($bytes, 2)
        for ($i = 0; $i -lt $sections; $i++) {
            $header = 20 + 40 * $i
            $name = [Text.Encoding]::ASCII.GetString($bytes, $header, 8).TrimEnd([char]0)
            if ($name -eq '.debug$S') {
                $size = [BitConverter]::ToUInt32($bytes, $header + 16)
                $offset = [BitConverter]::ToUInt32($bytes, $header + 20)
                [Array]::Clear($bytes, $offset, $size)
            }
        }
        # The symbol table: PointerToSymbolTable at 8, NumberOfSymbols at 12, 18-byte records
        # (short name at 0, SectionNumber at 12, NumberOfAuxSymbols at 17)
        $symbols = [BitConverter]::ToUInt32($bytes, 8)
        $count = [BitConverter]::ToUInt32($bytes, 12)
        for ($i = 0; $i -lt $count; $i += 1 + $bytes[$symbols + 18 * $i + 17]) {
            $symbol = $symbols + 18 * $i
            $name = [Text.Encoding]::ASCII.GetString($bytes, $symbol, 8)
            if ($name -eq '@comp.id' -and [BitConverter]::ToInt16($bytes, $symbol + 12) -eq -1) {
                [BitConverter]::GetBytes([int16]-2).CopyTo($bytes, $symbol + 12)
            }
        }
        [IO.File]::WriteAllBytes($out, $bytes)
        Write-Host "Wrote $out"
    }
} finally {
    Remove-Item -Recurse -Force $tmp
}
