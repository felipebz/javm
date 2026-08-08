@echo off
setlocal DisableDelayedExpansion
set "_JAVM_EXECUTABLE=::JAVM::"

if not exist "%_JAVM_EXECUTABLE%" (
    >&2 echo javm: executable not found at "%_JAVM_EXECUTABLE%". Regenerate this wrapper with "javm init cmd".
    endlocal & exit /b 9009
)

::JAVM_DEFAULT_INIT::

:javm_dispatch
if /i "%~1"=="use" goto javm_environment_command
if /i "%~1"=="deactivate" goto javm_environment_command

"%_JAVM_EXECUTABLE%" %*
set "_JAVM_EXIT_CODE=%ERRORLEVEL%"
endlocal & exit /b %_JAVM_EXIT_CODE%

:javm_initialize_default
set "_JAVM_APPLY_DEFAULT=1"
goto javm_environment_command

:javm_environment_command
set "_JAVM_TEMP_DIR=%TEMP%\javm-%RANDOM%-%RANDOM%-%RANDOM%"
2>nul mkdir "%_JAVM_TEMP_DIR%"
if errorlevel 1 goto javm_environment_command
set "_JAVM_ENV_FILE=%_JAVM_TEMP_DIR%\environment"

if defined _JAVM_APPLY_DEFAULT goto javm_run_default
"%_JAVM_EXECUTABLE%" --fd3 "%_JAVM_ENV_FILE%" %*
goto javm_environment_result

:javm_run_default
"%_JAVM_EXECUTABLE%" --fd3 "%_JAVM_ENV_FILE%" use --default

:javm_environment_result
set "_JAVM_EXIT_CODE=%ERRORLEVEL%"

if not "%_JAVM_EXIT_CODE%"=="0" if defined _JAVM_APPLY_DEFAULT goto javm_skip_default
if not "%_JAVM_EXIT_CODE%"=="0" goto javm_cleanup
if not exist "%_JAVM_ENV_FILE%" if defined _JAVM_APPLY_DEFAULT goto javm_skip_default
if not exist "%_JAVM_ENV_FILE%" goto javm_missing_environment

for /f "usebackq tokens=1,2,* delims=	" %%A in ("%_JAVM_ENV_FILE%") do (
    if /i "%%A"=="SET" if /i "%%B"=="PATH" set "_JAVM_NEW_PATH=%%C"
    if /i "%%A"=="SET" if /i "%%B"=="JAVA_HOME" set "_JAVM_NEW_JAVA_HOME=%%C"
    if /i "%%A"=="SET" if /i "%%B"=="JAVA_HOME_BEFORE_JAVM" set "_JAVM_NEW_JAVA_HOME_BEFORE_JAVM=%%C"
    if /i "%%A"=="UNSET" if /i "%%B"=="JAVA_HOME_BEFORE_JAVM" set "_JAVM_UNSET_JAVA_HOME_BEFORE_JAVM=1"
)

if defined _JAVM_APPLY_DEFAULT goto javm_apply_default
if /i "%~1"=="use" goto javm_apply_use
goto javm_apply_deactivate

:javm_missing_environment
>&2 echo javm: environment update file was not created.
set "_JAVM_EXIT_CODE=1"

:javm_cleanup
del /q "%_JAVM_ENV_FILE%" >nul 2>&1
rmdir "%_JAVM_TEMP_DIR%" >nul 2>&1
endlocal & exit /b %_JAVM_EXIT_CODE%

:javm_skip_default
del /q "%_JAVM_ENV_FILE%" >nul 2>&1
rmdir "%_JAVM_TEMP_DIR%" >nul 2>&1
endlocal & set "_JAVM_DEFAULT_INITIALIZED=1"
goto javm_after_default

:javm_apply_default
del /q "%_JAVM_ENV_FILE%" >nul 2>&1
rmdir "%_JAVM_TEMP_DIR%" >nul 2>&1
endlocal & set "PATH=%_JAVM_NEW_PATH%" & set "JAVA_HOME=%_JAVM_NEW_JAVA_HOME%" & set "JAVA_HOME_BEFORE_JAVM=%_JAVM_NEW_JAVA_HOME_BEFORE_JAVM%" & set "_JAVM_DEFAULT_INITIALIZED=1"

:javm_after_default
setlocal DisableDelayedExpansion
set "_JAVM_EXECUTABLE=::JAVM::"
goto javm_dispatch

:javm_apply_use
del /q "%_JAVM_ENV_FILE%" >nul 2>&1
rmdir "%_JAVM_TEMP_DIR%" >nul 2>&1
endlocal & set "PATH=%_JAVM_NEW_PATH%" & set "JAVA_HOME=%_JAVM_NEW_JAVA_HOME%" & set "JAVA_HOME_BEFORE_JAVM=%_JAVM_NEW_JAVA_HOME_BEFORE_JAVM%" & set "_JAVM_DEFAULT_INITIALIZED=1" & exit /b %_JAVM_EXIT_CODE%

:javm_apply_deactivate
del /q "%_JAVM_ENV_FILE%" >nul 2>&1
rmdir "%_JAVM_TEMP_DIR%" >nul 2>&1
endlocal & set "PATH=%_JAVM_NEW_PATH%" & set "JAVA_HOME=%_JAVM_NEW_JAVA_HOME%" & set "JAVA_HOME_BEFORE_JAVM=" & set "_JAVM_DEFAULT_INITIALIZED=1" & exit /b %_JAVM_EXIT_CODE%
