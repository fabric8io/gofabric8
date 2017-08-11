:: Run "set F8_DEBUG=1" before calling this script to enable debug output.
:: This basically just outputs every command before executing it.
@if not defined F8_DEBUG echo off

:: This is a script to help have a CI system on a Windows host with no
:: other requirement other than to have a CMD.

:: Leave this line here as it avoids error with "input line is too long"
:: due to too long PATH environment variable.
setlocal enableextensions enabledelayedexpansion

:: Remember program name (regardless of calling method) for functions to use
set PROG_NAME=%~nx0
set TARGET=%1
if not "%PROCESSOR_ARCHITECTURE%" == "AMD64" goto error_wrong_architecture

:: The workspace environment is set by Jenkins and defaults to %TEMP% if not set
if "%WORKSPACE%" == "" set WORKSPACE=%TEMP%
if "%BUILD_DIR%" == "" set "BUILD_DIR=%WORKSPACE%\fabric8-wit-windows-build"
:: Create our build dir - see https://support.microsoft.com/en-us/kb/65994
if not exist %BUILD_DIR%\NUL (
  mkdir %BUILD_DIR%
  if %errorlevel% neq 0 goto error_build_dir
)
set SOURCE_DIR=%CD%\..\..
set SCRIPT_ROOT_DIR=%CD%
set POWERSHELL_BIN=%SystemRoot%\system32\WindowsPowerShell\v1.0\powershell.exe
set GOPATH=%BUILD_DIR%\gopath
set GOBIN=%GOPATH%\bin
set PATH=%PATH%;%GOBIN%
set PACKAGE_PATH=%GOPATH%\src\github.com\fabric8-services\fabric8-wit

set GIT_DOWNLOAD_URL=https://github.com/git-for-windows/git/releases/download/v2.9.0.windows.1/Git-2.9.0-64-bit.exe
set GIT_INSTALLER_PATH=%BUILD_DIR%\git-installer.exe
set GIT_INSTALL_PREFIX=%BUILD_DIR%\git-install-dir
set "PATH=%PATH%;%GIT_INSTALL_PREFIX%\bin"

set MERCURIAL_DOWNLOAD_URL=https://www.mercurial-scm.org/release/windows/Mercurial-3.8.4-x64.exe
set MERCURIAL_INSTALLER_PATH=%BUILD_DIR%\mercurial-installer.exe
set MERCURIAL_INSTALL_PREFIX=%BUILD_DIR%\mercurial-install-dir
set "PATH=%PATH%;%MERCURIAL_INSTALL_PREFIX%"

set GLIDE_DOWNLOAD_URL=https://github.com/Masterminds/glide/releases/download/v0.11.0/glide-v0.11.0-windows-amd64.zip
set GLIDE_ZIP_PATH=%BUILD_DIR%\glide.zip
set GLIDE_INSTALL_PREFIX=%BUILD_DIR%\glide-install-dir
set "PATH=%PATH%;%GLIDE_INSTALL_PREFIX%\windows-amd64"

set GO_DOWNLOAD_URL=https://storage.googleapis.com/golang/go1.6.2.windows-amd64.zip
set GO_ZIP_PATH=%BUILD_DIR%\go.zip
set GO_INSTALL_PREFIX=%BUILD_DIR%\go-install-dir
set "GOROOT=%GO_INSTALL_PREFIX%\go"
set "PATH=%PATH%;%BUILD_DIR%\go-install-dir\go\bin"

set CYGWIN_PACKAGES=make
set CYGWIN_DOWNLOAD_URL=https://cygwin.com/setup-x86_64.exe
set CYGWIN_INSTALLER_PATH=%BUILD_DIR%\cygwin-installer.exe
set CYGWIN_INSTALL_PREFIX=%BUILD_DIR%\cygwin-install-dir
set CYGWIN_PACKAGE_PATH=%BUILD_DIR%\cygwin-package-dir
set CYGWIN_PACKAGE_SITE_URL=http://cygwin.mirror.constant.com
:: The cygwin path MUST be before the Windows path because otherwise tools
:: like "find" will be taken from the Windows version which won't work.
set "PATH=%CYGWIN_INSTALL_PREFIX%\bin;%PATH%"


:: -----------------------------------------------------------------
:: Dispatch the make target
:: -----------------------------------------------------------------

if "%1" == "" ( call :help )
if "%TARGET%" == "help"                      call :help )
if "%TARGET%" == "setup"                     call :setup )
if "%TARGET%" == "copy-source-to-build-dir"  call :copy-source-to-build-dir
if "%TARGET%" == "deps"                      call :deps )
if "%TARGET%" == "generate"                  call :generate )
if "%TARGET%" == "build"                     call :build )
if "%TARGET%" == "dev"                       call :dev )
if "%TARGET%" == "clean"                     call :clean )
if "%TARGET%" == "clean-vendor"              call :clean-vendor )
if "%TARGET%" == "wipe-out"                  call :wipe-out )
if "%TARGET%" == "wipe-out-package-path"     call :wipe-out-package-path )
if "%TARGET%" == "test"                      call :test )
if "%TARGET%" == "jenkins"                   call :jenkins )

:: advanced targets

if "%TARGET%" == "cygwin-download-installer" call :cygwin-download-installer )
if "%TARGET%" == "cygwin-download-packages"  call :cygwin-download-packages )
if "%TARGET%" == "cygwin-install-packages"   call :cygwin-install-packages )
if "%TARGET%" == "cygwin-clean"              call :cygwin-clean )

if "%TARGET%" == "git-download"              call :git-download )
if "%TARGET%" == "git-install"               call :git-install )
if "%TARGET%" == "git-clean"                 call :git-clean )

if "%TARGET%" == "mercurial-download"        call :mercurial-download )
if "%TARGET%" == "mercurial-install"         call :mercurial-install )
if "%TARGET%" == "mercurial-clean"           call :mercurial-clean )

if "%TARGET%" == "glide-download"            call :glide-download )
if "%TARGET%" == "glide-install"             call :glide-install )
if "%TARGET%" == "glide-clean"               call :glide-clean )

if "%TARGET%" == "go-download"               call :go-download )
if "%TARGET%" == "go-install"                call :go-install )
if "%TARGET%" == "go-clean"                  call :go-clean )

goto end

:jenkins
call :infolog Running jenkins target
call :setup
if %errorlevel% neq 0 (
	echo ERROR: Setup finished with errors
	goto end
)
call :wipe-out-package-path
if %errorlevel% neq 0 (
	echo ERROR: Failed to wipe out the package path %PACKAGE_PATH%
	goto end
)
call :copy-source-to-build-dir
if %errorlevel% neq 0 (
	echo ERROR: Failed to copy source to build dir
	goto end
)
call :deps
if %errorlevel% neq 0 (
	echo ERROR: Failed to fetch dependencies
	goto end
)
call :generate
if %errorlevel% neq 0 (
	echo ERROR: Failed to generate go source with GOA
	goto end
)
call :build
if %errorlevel% neq 0 (
	echo ERROR: Failed to build
	goto end
)
call :test
if %errorlevel% neq 0 (
	echo ERROR: Failed to run tests
	goto end
)
exit /B %errorlevel

:setup
call :infolog Running setup
:: check if we need to install cygwin
if not exist %CYGWIN_INSTALL_PREFIX%\NUL (
	if not exist %CYGWIN_INSTALLER_PATH% call :cygwin-download-installer
	if not exist %CYGWIN_PACKAGE_PATH%\NUL call :cygwin-download-packages
	call :cygwin-install-packages
) else (
	echo Cygwin is already set up
)
:: check if we need to install git
if not exist %GIT_INSTALL_PREFIX%\NUL (
	if not exist %GIT_INSTALLER_PATH% call :git-download
	call :git-install
) else (
	echo Git is already set up
)
:: check if we need to install mercurial
if not exist %MERCURIAL_INSTALL_PREFIX%\NUL (
	if not exist %MERCURIAL_INSTALLER_PATH% call :mercurial-download
	call :mercurial-install
) else (
	echo Mercurial is already set up
)
:: check if we need to install glide
if not exist %GLIDE_INSTALL_PREFIX%\NUL (
	if not exist %GLIDE_ZIP_PATH% call :glide-download
	call :glide-install
) else (
	echo Glide is already set up
)
:: check if we need to install go
if not exist %GO_INSTALL_PREFIX%\NUL  (
	if not exist %GO_ZIP_PATH% call :go-download
	call :go-install
) else (
	echo Go is already set up
)
call :infolog Setup is done
exit /B 0

:clean-vendor
call :infolog Removing the directory with Go dependencies: %PACKAGE_PATH%\vendor
:: Removing an existing vendor dir safes us from trouble ;)
if exist %PACKAGE_PATH%\vendor\NUL rmdir /S /Q %PACKAGE_PATH%\vendor
exit /B 0

:copy-source-to-build-dir
call :infolog Copy source to build dir
if not exist %PACKAGE_PATH%\NUL mkdir %PACKAGE_PATH%
:: see https://technet.microsoft.com/de-de/library/cc733145(v=ws.10).aspx for options on robocopy
robocopy %SOURCE_DIR% %PACKAGE_PATH% /E /copyall /purge
:: TODO: robocopy seems to run in false positive (?) errors a lot so we cannot exit with %errorlevel% here. Fix this
@exit /B 0

:wipe-out
call :infolog Wiping out build dir %BUILD_DIR%
:: navigate out of the build dir before we can delete it
cd %SCRIPT_ROOT%
if exist %BUILD_DIR%\NUL rmdir /S /Q "%BUILD_DIR%"
exit /B %errorlevel%

:wipe-out-package-path
call :infolog Wiping out the package-path %PACKAGE_PATH%
:: navigate out of the package dir before we can delete it
cd %SCRIPT_ROOT%
if exist %PACKAGE_PATH%\NUL rmdir /S /Q "%PACKAGE_PATH%"
exit /B %errorlevel%

:: ---------------------------------------------------------------------
:: Mappings to original Makefile
:: ---------------------------------------------------------------------

:deps
call :infolog Installing Go dependencies using glide
cd %PACKAGE_PATH%
:: This local PATH adjustments are so that not the git.exe o
:: hg.exe are being used from cygwin when running glide.
setlocal
set "PATH=%GIT_INSTALL_PREFIX%\bin;%PATH%"
set "PATH=%MERCURIAL_INSTALL_PREFIX%\bin;%PATH%"
glide install
endlocal
@exit /B %errorlevel%

:generate
call :infolog Generating code
cd %PACKAGE_PATH%
bash -x -c "export SOURCE_DIR=$(cygpath --unix '%PACKAGE_PATH%'); make generate"
exit /B %errorlevel%

:build
call :infolog Building
cd %PACKAGE_PATH%
bash -x -c "export SOURCE_DIR=$(cygpath --unix '%PACKAGE_PATH%'); make build"
exit /B %errorlevel%

:clean
call :infolog Cleaning up
cd %PACKAGE_PATH%
bash -x -c "export SOURCE_DIR=$(cygpath --unix '%PACKAGE_PATH%'); make clean"
exit /B %errorlevel%

:dev
call :infolog Spawning developer environment
cd %PACKAGE_PATH%
bash -x -c "export SOURCE_DIR=$(cygpath --unix '%PACKAGE_PATH%'); make dev"
exit /B %errorlevel%

:test
call :infolog Running tests
cd %PACKAGE_PATH%
bash -x -c "export SOURCE_DIR=$(cygpath --unix '%PACKAGE_PATH%'); make test"
exit /B %errorlevel%

:: ---------------------------------------------------------------------
:: Cygwin
:: ---------------------------------------------------------------------

:cygwin-download-installer
call :infolog Downloading Cygwin installer
call :download-file %CYGWIN_DOWNLOAD_URL% %CYGWIN_INSTALLER_PATH%
exit /B 0

:cygwin-download-packages
call :infolog Downloading Cygwin packages to %CYGWIN_PACKAGE_PATH%
%CYGWIN_INSTALLER_PATH% --no-admin --arch=x86_64 --packages=git ^
	--root=%CYGWIN_INSTALL_PREFIX% --no-shortcuts --no-verify ^
	--site=%CYGWIN_PACKAGE_SITE_URL% --quiet-mode ^
	--no-verify --local-package-dir=%CYGWIN_PACKAGE_PATH% ^
	--packages=%CYGWIN_PACKAGES% ^
	--download
if %errorlevel% neq 0 goto error_cygwin_download_packages
exit /B 0

:cygwin-install-packages
call :infolog Installing Cygwin packages from %CYGWIN_PACKAGE_PATH% to %CYGWIN_INSTALL_PREFIX%
%CYGWIN_INSTALLER_PATH% --no-admin --arch=x86_64 --packages=git ^
	--root=%CYGWIN_INSTALL_PREFIX% --no-shortcuts --no-verify ^
	--site=%CYGWIN_PACKAGE_SITE_URL% --quiet-mode ^
	--no-verify --local-package-dir=%CYGWIN_PACKAGE_PATH% ^
	--packages=%CYGWIN_PACKAGES% ^
	--local-install
if %errorlevel% neq 0 goto error_cygwin_install_packages
exit /B 0

:cygwin-clean
call :infolog Cleaning up cygwin
if exist %CYGWIN_INSTALLER_PATH% ( del %CYGWIN_INSTALLER_PATH% )
if exist %CYGWIN_INSTALL_PREFIX% ( rmdir /S /Q %CYGWIN_INSTALL_PREFIX% )
if exist %CYGWIN_PACKAGE_PATH% ( rmdir /S /Q %CYGWIN_PACKAGE_PATH% )
exit /B 0

:: ---------------------------------------------------------------------
:: Git
:: ---------------------------------------------------------------------

:git-download
call :infolog Downloading Git
call :download-file %GIT_DOWNLOAD_URL% %GIT_INSTALLER_PATH%
exit /B 0

:git-install
call :infolog Installing Git
if not exist %GIT_INSTALL_PREFIX% (
	echo Installing Git from %GIT_INSTALLER_PATH% to %GIT_INSTALL_PREFIX%
	:: See http://www.jrsoftware.org/ishelp/index.php?topic=setupcmdline for options
	%GIT_INSTALLER_PATH% /SP- /SUPPRESSMSGBOXES /NORESTART ^
		/NOCLOSEAPPLICATIONS /NORESTARTAPPLICATIONS /DIR=%GIT_INSTALL_PREFIX% /NOICONS ^
		/LOADINF="%SOURCE_DIR%\jenkins\windows\git.inf" /SILENT /VERYSILENT
	if %errorlevel% neq 0 goto error_git_install
) else (
	echo Git is already installed in %GIT_INSTALL_PREFIX%
)
exit /B 0

:git-clean
call :infolog Cleaning up Git
if exist %GIT_INSTALLER_PATH% ( del %GIT_INSTALLER_PATH% )
if exist %GIT_INSTALL_PREFIX% ( rmdir /S /Q %GIT_INSTALL_PREFIX% )
exit /B 0

:: ---------------------------------------------------------------------
:: Mercurial
:: ---------------------------------------------------------------------

:mercurial-download
call :infolog Downloading Mercurial
call :download-file %MERCURIAL_DOWNLOAD_URL% %MERCURIAL_INSTALLER_PATH%
exit /B 0

:mercurial-install
call :infolog Installing Mercurial from %MERCURIAL_INSTALLER_PATH% to %MERCURIAL_INSTALL_PREFIX%
if not exist %MERCURIAL_INSTALL_PREFIX% (
	:: See http://www.jrsoftware.org/ishelp/index.php?topic=setupcmdline for options
	%MERCURIAL_INSTALLER_PATH% /SP- /SUPPRESSMSGBOXES /NORESTART ^
		/NOCLOSEAPPLICATIONS /NORESTARTAPPLICATIONS /DIR=%MERCURIAL_INSTALL_PREFIX% /NOICONS ^
		/LOADINF="%SOURCE_DIR%\jenkins\windows\mercurial.inf" /SILENT /VERYSILENT
	if %errorlevel% neq 0 goto error_mercurial_install
) else (
	echo Mercurial is already installed in %MERCURIAL_INSTALL_PREFIX%
)
exit /B 0

:mercurial-clean
call :infolog Cleaning up Mercurial
if exist %MERCURIAL_INSTALLER_PATH% ( del %MERCURIAL_INSTALLER_PATH% )
if exist %MERCURIAL_INSTALL_PREFIX% ( rmdir /S /Q %MERCURIAL_INSTALL_PREFIX% )
exit /B 0

:: ---------------------------------------------------------------------
:: Glide
:: ---------------------------------------------------------------------

:glide-download
call :infolog Downloading Glide
call :download-file %GLIDE_DOWNLOAD_URL% %GLIDE_ZIP_PATH%
exit /B 0

:glide-install
call :infolog Installing Glide
call :extract-file %GLIDE_ZIP_PATH% %GLIDE_INSTALL_PREFIX%
exit /B 0

:glide-clean
call :infolog Cleaning up Glide
if exist %GLIDE_ZIP_PATH% ( del %GLIDE_ZIP_PATH% )
if exist %GLIDE_INSTALL_PREFIX% ( rmdir /S /Q %GLIDE_INSTALL_PREFIX% )
exit /B 0

:: ---------------------------------------------------------------------
:: Go
:: ---------------------------------------------------------------------

:go-download
call :infolog Downloading Golang
call :download-file %GO_DOWNLOAD_URL% %GO_ZIP_PATH%
exit /B 0

:go-install
call :infolog Installing Golang
call :extract-file %GO_ZIP_PATH% %GO_INSTALL_PREFIX%
mkdir "%GOPATH%"
mkdir "%GOPATH%\bin"
mkdir "%GOPATH%\pkg"
mkdir "%GOPATH%\src"
exit /B 0

:go-clean
call :infolog Cleaning up Golang
if exist %GO_ZIP_NAME% ( del %GO_ZIP_NAME% )
if exist %GO_INSTALL_PREFIX% ( rmdir /S /Q %GO_INSTALL_PREFIX% )
exit /B 0

:: ---------------------------------------------------------------------
:: Helper functions
:: ---------------------------------------------------------------------

:infolog
:: Writes all arguments to the screen prefixed with a little text
:: and sets the title text for the command window.
echo.[INFO] [%PROG_NAME%] %*
title %*
exit /B 0

:download-file
:: Downloads a URL (1st argument) to a local file (2nd argument).
:: If the file already exists on disk it will not be downloaded
:: but a message is written to the screen.
if not exist %2 (
	%POWERSHELL_BIN% -ExecutionPolicy remotesigned -File %SCRIPT_ROOT_DIR%\download-file.ps1 -url %1 -file %2
	if %errorlevel% neq 0 goto error_download_file
) else (
	echo Target file %2 already exists
)
exit /B 0

:extract-file
:: Unpacks a file (1st argument) into the directory (2nd argument).
if not exist %2 (
	%POWERSHELL_BIN% -ExecutionPolicy remotesigned -File %SCRIPT_ROOT_DIR%\extract-zip.ps1 -file %1 -prefix %2
	if %errorlevel% neq 0 goto error_extract_file
) else (
	echo Target directory %2 already exists
)
exit /B 0

:help
call :infolog Show help for %PROG_NAME%
echo.
echo Synopsis:
echo ---------
echo This tool can be used on Windows to help setup a build environment and build
echo the software. All the tools needed to build the software can be installed on
echo request in the right version and without interfering with the rest of the
echo operating system.
echo.
echo This tool builds the software in %BUILD_DIR% unless you tell it otherwise
echo using the BUILD_DIR environment variable.
echo.
echo Usage:
echo ------
echo.
echo %PROG_NAME% [TARGET]
echo.
echo TARGET can be one of the following:
echo -----------------------------------
echo.
echo help                      - show this usage information
echo setup                     - download and setup all tools needed (e.g. cygwin, go, git, mercurial, and glide)
echo copy-source-to-build-dir  - copies complete source code dir over to %PACKAGE_PATH%
echo deps                      - fetches the Go dependencies with glide
echo generate                  - generates Go files from designs
echo build                     - builds the client and server artifacts
echo test                      - executes tests
echo clean                     - removes build artifacts, generated and vendored code from %PACKAGE_PATH%
echo clean-vendor              - just removes the vendored Go dependencies directory in %PACKAGE_PATH%\vendor
echo wipe-out                  - wipes out the complete build dir %BUILD_DIR%
echo wipe-out-package-path     - wipes out the copy of source code inside of %PACKAGE_PATH%
echo jenkins                   - Runs all the things that jenkins needs to execute
echo.
echo Advanced targets:
echo -----------------
echo.
echo cygwin-download-installer - downloads cygwin installer from %CYGWIN_DOWNLOAD_URL% to %CYGWIN_INSTALLER_PATH%
echo cygwin-download-packages  - downloads cygwin packages from %CYGWIN_PACKAGE_SITE_URL% to %CYGWIN_PACKAGE_PATH%
echo cygwin-install-packages   - installs cygwin packages from %CYGWIN_PACKAGE_PATH% to %CYGWIN_INSTALL_PREFIX%
echo cygwin-clean              - removes the cygwin installer, any downloaded packages, and the installation directory for cygwin
echo.
echo git-download              - downloads git installer from %GIT_DOWNLOAD_URL% to %GIT_INSTALLER_PATH%
echo git-install               - unpacks git from %GIT_INSTALLER_PATH% to %GIT_INSTALL_PREFIX%
echo git-clean                 - removes %GIT_INSTALL_PREFIX% and %GIT_INSTALLER_PATH%
echo.
echo mercurial-download        - downloads mercurial installer from %MERCURIAL_DOWNLOAD_URL% to %MERCURIAL_INSTALLER_PATH%
echo mercurial-install         - unpacks mercurial from %MERCURIAL_INSTALLER_PATH% to %MERCURIAL_INSTALL_PREFIX%
echo mercurial-clean           - removes %MERCURIAL_INSTALL_PREFIX% and %MERCURIAL_INSTALLER_PATH%
echo.
echo glide-download            - downloads glide zip from %GLIDE_DOWNLOAD_URL% to %GLIDE_ZIP_PATH%
echo glide-install             - unpacks glide from %GLIDE_ZIP_PATH% to %GLIDE_INSTALL_PREFIX%
echo glide-clean               - removes %GLIDE_INSTALL_PREFIX% and %GLIDE_ZIP_PATH%
echo.
echo go-download               - downloads go zip from %GO_DOWNLOAD_URL% to %GO_ZIP_PATH%
echo go-install                - unpacks go from %GO_ZIP_PATH% to %GO_INSTALL_PREFIX%
echo go-clean                  - removes %GO_INSTALL_PREFIX% and %GO_ZIP_PATH%
echo.
@exit /B 0

:: ---------------------------------------------------------------------
:: Errors
:: ---------------------------------------------------------------------

:error_wrong_architecture
echo ERROR: The script %PROG_NAME% requires a 64 bit operating system
goto end

:error_build_dir
echo ERROR: Failed to create the build dir %BUILD_DIR%
goto end

:error_download_file
echo ERROR: Download failed
goto end

:error_extract_file
echo ERROR: Failed to extract file
goto end

:error_cygwin_download_packages
echo ERROR: Cygwin download packages failed
goto end

:error_cygwin_install_packages
echo ERROR: Cygwin installation of packages failed
goto end

:error_glide_install
echo ERROR: Glide installation failed
goto end

:error_git_install
echo ERROR: Git installation failed
goto end

:error_mercurial_install
echo ERROR: Mercurial installation failed
goto end

:error_go_install
echo ERROR: Golang installation failed
goto end

:end
	echo Exiting %PROG_NAME% with error code %errorlevel% (0 = success, 1 = error, 2 = unknown)
	@exit /b %errorlevel%
