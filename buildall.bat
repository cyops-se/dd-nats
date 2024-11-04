@echo off

echo Building UI
call :buildUI

echo Building all executables for: %GOOS%, arch: %GOARCH%
call :treeProcess

echo Copying executables to dist folder dist\%GOOS%
call :makeDist
goto :eof

:buildUI
cd ui
call npm i 
call npm run build 
rmdir /q /s ..\svcs\dd-ui\web\static
xcopy /e /y /q dist\* ..\svcs\dd-ui\web\static\
cd ..
exit /b

:treeProcess
rem Do whatever you want here over the files of this subdir, for example:
for %%f in (build*.bat) do (
    if "%%f" NEQ "buildall.bat" call %%f
)

for /D %%d in (*) do (
    if "%%d" NEQ "node_modules" (
        cd %%d
        call :treeProcess
        cd ..
    )
)
exit /b

:makeDist
if "%GOOS%" == "" set GOOS=windows
if "%GOOS%" == "windows" (
    rmdir /q /s .\dist\%GOOS%
    xcopy /q /y .\inner\dd-nats-cache\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\inner\dd-nats-file-inner\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\inner\dd-nats-inner-proxy\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\inner\dd-nats-modbus\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\inner\dd-nats-opcda\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\tools\sniffer\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\svcs\dd-logger\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\svcs\dd-ui\*.exe .\dist\%GOOS%\inner\
    xcopy /q /y .\outer\dd-nats-cache-unpack\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-file-outer\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-influxdb\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-outer-proxy\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-process-filter\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-rabbitmq\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-timescale\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\tools\sniffer\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\svcs\dd-logger\*.exe .\dist\%GOOS%\outer\
    xcopy /q /y .\svcs\dd-ui\*.exe .\dist\%GOOS%\outer\
)

if "%GOOS%" == "linux" (
    rmdir /q /s .\dist\%GOOS%
    xcopy /q /y .\inner\dd-nats-cache\dd-nats-cache .\dist\%GOOS%\inner\
    xcopy /q /y .\inner\dd-nats-file-inner\dd-nats-file-inner .\dist\%GOOS%\inner\
    xcopy /q /y .\inner\dd-nats-inner-proxy\dd-nats-inner-proxy .\dist\%GOOS%\inner\
    xcopy /q /y .\inner\dd-nats-modbus\dd-nats-modbus .\dist\%GOOS%\inner\
    xcopy /q /y .\tools\sniffer\sniffer .\dist\%GOOS%\inner\
    xcopy /q /y .\svcs\dd-logger\dd-logger .\dist\%GOOS%\inner\
    xcopy /q /y .\svcs\dd-ui\dd-ui .\dist\%GOOS%\inner\
    xcopy /q /y .\outer\dd-nats-cache-unpack\dd-nats-cache-unpack .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-file-outer\dd-nats-file-outer .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-influxdb\dd-nats-influxdb .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-outer-proxy\dd-nats-outer-proxy .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-process-filter\dd-nats-process-filter .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-rabbitmq\dd-nats-rabbitmq .\dist\%GOOS%\outer\
    xcopy /q /y .\outer\dd-nats-timescale\dd-nats-timescale .\dist\%GOOS%\outer\
    xcopy /q /y .\tools\sniffer\sniffer .\dist\%GOOS%\outer\
    xcopy /q /y .\svcs\dd-logger\dd-logger .\dist\%GOOS%\outer\
    xcopy /q /y .\svcs\dd-ui\dd-ui .\dist\%GOOS%\outer\
)
exit /b

:eof
