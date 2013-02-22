@echo off
:: "Make Go" v1.06 2013-02-22
:: Tool for easy building Go (http://golang.org) programs out of the
:: main %GOPATH% tree.
:: SOURCE: https://gist.github.com/3713260
::
:: TODO:
:: - add subcommands, especially: install, clean (rmdir /q /s _gopath)
:: - add info that when importing, you must prefix with current dir name
:: OTHER IDEAS:
:: - port the tool to Go?
::
:: If confused with the code, consult the indispensable ss64.com site.
::
:: Code is in public domain, where available. For the rest of the world, choose BSD license or MIT/X11 one, 
:: whichever you prefer.
setlocal
set errorlevel=
set dir0="%cd%"

if "%1"=="help" goto help
if "%1"=="-help" goto help
if "%1"=="--help" goto help
if "%1"=="-h" goto help
if "%1"=="/?" goto help
if "%1"=="/h" goto help
goto nohelp
:help
echo USAGE: mg [COMMAND] [PACKAGE]
echo - COMMAND: local [default], test, run, help
echo - PACKAGE: Go package name, default: "."
echo.
echo Copies contents of current dir to fake GOPATH, then runs 'go' there.
echo When copying, directories starting with underscore "_" are skipped.
echo.
echo NOTE: You can put additional files with options in current dir:
echo - 'goroot.txt' - contents will be put in GOROOT variable,
echo and resulting GOROOT\bin to PATH.
echo - 'goname.txt' - contents will be used as full import path
echo of your package.
echo - 'gopath.txt' - contents will be appended to GOPATH [don't use quotes].
goto :end
:nohelp

:: find out what subcommand is called, or use default
if "%1"=="test" set func=%1
if "%1"=="help" set func=%1
if "%1"=="run"  set func=%1
if "%func%"=="" (
	:: default subcommand
	set func=local
) else (
	:: delete the subcommand from arguments list
	shift
)
call :func_%func%

:: find out if a package path was provided
set gopackage=.
if not "%1"=="" (
	set gopackage=%1
)

:: load GOROOT if supplied
if not exist goroot.txt goto :nogoroot
	set /p GOROOT=<goroot.txt
	set PATH=%GOROOT%\bin;%PATH%
:nogoroot

if not exist goname.txt goto :nogoname
	set /p projdir=<goname.txt
	set projdir=_gopath\src\"%projdir:/=\%"
	goto :hasgoname
:nogoname
	call :basename "%cd%"
	set projdir=_gopath\src\"%_result%"
:hasgoname

if exist %projdir% (
	rmdir /s /q %projdir% >nul
	rmdir /q %projdir% >nul 2>&1
)
mkdir %projdir% >nul
if errorlevel 1 (
	echo cannot mkdir %projdir%
	goto :end
)

:: copy subdirs, except those starting with underscore '_'
for /f "tokens=*" %%f in ('dir /b /ad') do call :copydir %%f
goto :endcopydir
:copydir
	set fn=%*
	if not "%fn:~0,1%"=="_" (
		mkdir %projdir%\"%fn%" >nul
		xcopy /q /e "%fn%"\*.* %projdir%\"%fn%"\ >nul
		if errorlevel 1 (
			echo cannot xcopy /q /e "%fn%"\*.* %projdir%\"%fn%"\
			goto :end
		)
	)
	goto :eof
:endcopydir

:: copy files
for %%f in (*.*) do (
	copy "%%f" %projdir%\ >nul
	if errorlevel 1 (
		echo cannot copy "%%f" %projdir%\
		goto :end
	)
)

:: note: gopath doesn't want quotes, but seems to work OK with spaces
if not exist gopath.txt goto nogopath
	set /p gopath=<gopath.txt
:nogopath
set gopath=%cd%\_gopath;%gopath%
::echo %gopath%

cd %projdir%
if "%func%"=="test" (
	go.exe test -c %gopackage%
	go.exe test -v %gopackage%
	goto :end
)
go.exe %f_gocmd% %opts% %gopackage%
goto :end

:func_local
set f_gocmd=install
goto :eof

:func_test
set f_gocmd=test
goto :eof

:func_run
set f_gocmd=run
goto :eof

:basename %1
set _result=%~n1
goto :eof

:end
set _err=%errorlevel%
cd %dir0%
endlocal && cmd /c "exit %_err%"
