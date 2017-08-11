# Downloads a file from $url and saves it to $file
param (
	[Parameter(Mandatory=$true)][string]$url,
	[Parameter(Mandatory=$true)][string]$file
)

try {

	# Download
	Write-Output "Downloading $url to $file"
	$start_time = Get-Date
	$wc = New-Object System.Net.WebClient
	$wc.DownloadFile($url, $file)
	Write-Output "Time taken: $((Get-Date).Subtract($start_time).Seconds) second(s)"
}
catch {
	$error[0] | fl -force
}
