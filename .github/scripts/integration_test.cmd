@echo off
setlocal EnableExtensions DisableDelayedExpansion

if not defined JAVA_VERSION set "JAVA_VERSION=21"

echo ^>^>^> Exercising discovery/ls-remote
call javm ls-remote 21 || exit /b 1
call javm ls --details || exit /b 1

echo ^>^>^> Installing JDK %JAVA_VERSION% ^(idempotent^)
call javm install "%JAVA_VERSION%" || exit /b 1

set "JAVA_HOME_BEFORE_CMD_TEST=%JAVA_HOME%"
echo ^>^>^> Using JDK %JAVA_VERSION%
call javm use "%JAVA_VERSION%" || exit /b 1

if not defined JAVA_HOME (
    >&2 echo JAVA_HOME is empty
    exit /b 1
)
if not exist "%JAVA_HOME%\bin\java.exe" (
    >&2 echo java.exe not found in "%JAVA_HOME%\bin"
    exit /b 1
)

where java || exit /b 1
java --version || exit /b 1

for /f "usebackq delims=" %%H in (`call javm which "%JAVA_VERSION%" --home`) do set "EXPECTED_JAVA_HOME=%%H"
if /i not "%JAVA_HOME%"=="%EXPECTED_JAVA_HOME%" (
    >&2 echo JAVA_HOME "%JAVA_HOME%" does not match javm which "%EXPECTED_JAVA_HOME%"
    exit /b 1
)

echo ^>^>^> Deactivating JDK
call javm deactivate || exit /b 1
if not "%JAVA_HOME%"=="%JAVA_HOME_BEFORE_CMD_TEST%" (
    >&2 echo JAVA_HOME was not restored
    exit /b 1
)

echo ^>^>^> CMD integration OK
exit /b 0
