# Extracts $file to $prefix
param (
	[Parameter(Mandatory=$true)][string]$file,
	[Parameter(Mandatory=$true)][string]$prefix
)

try {
	# Extract
	Write-Output "Extracting $file to directory $prefix"
	$start_time = Get-Date
	Add-Type -assembly "system.io.compression.filesystem"
	[io.compression.zipfile]::ExtractToDirectory($file, $prefix)
	Write-Output "Time taken: $((Get-Date).Subtract($start_time).Seconds) second(s)"
}
catch {
	$error[0] | fl -force
}
