$ErrorActionPreference = 'Stop'
$baseURL = 'http://127.0.0.1:8080'

for ($attempt = 0; $attempt -lt 60; $attempt++) {
    try {
        $ready = Invoke-RestMethod -Uri "$baseURL/readyz" -TimeoutSec 2
        if ($ready.status -eq 'ready') { break }
    } catch {}
    Start-Sleep -Seconds 2
}
if ($ready.status -ne 'ready') { throw 'LogPilot did not become ready' }

$runbook = Join-Path $PSScriptRoot '..\testdata\runbooks\go-panic.md'
$uploadRaw = & curl.exe -sS -f -F "file=@$runbook" -F "title=Go Panic Runbook" "$baseURL/api/v1/documents"
if ($LASTEXITCODE -ne 0) { throw 'Document upload request failed' }
$upload = $uploadRaw | ConvertFrom-Json
if (-not $upload.document_id) { throw 'Document upload failed' }

$body = @{ question = 'Why did the Go service hit a nil pointer panic?'; logs = 'ERROR panic: runtime error: invalid memory address' } | ConvertTo-Json
$diagnosis = Invoke-RestMethod -Method Post -Uri "$baseURL/api/v1/diagnoses" -ContentType 'application/json' -Body $body -TimeoutSec 90
if (-not $diagnosis.session_id -or -not $diagnosis.summary) { throw 'Diagnosis failed' }

$session = Invoke-RestMethod -Uri "$baseURL/api/v1/sessions/$($diagnosis.session_id)"
if ($session.messages.Count -lt 2) { throw 'Session history is incomplete' }

Write-Output 'SMOKE TEST PASSED'
