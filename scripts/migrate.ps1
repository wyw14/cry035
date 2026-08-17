param([switch]$Down)
$ErrorActionPreference = 'Stop'
if (-not $env:DATABASE_URL) { throw 'DATABASE_URL is required' }
if ($Down) { go run ./cmd/migrate down } else { go run ./cmd/migrate }
