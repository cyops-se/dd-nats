@echo off
call :treeProcess
goto :eof

:treeProcess
rem Do whatever you want here over the files of this subdir, for example:
for %%f in (*.exe) do start cmd /k %%f
for /D %%d in (*) do (
    if "%%d" NEQ "node_modules" (
        cd %%d
        call :treeProcess
        cd ..
    )
)
exit /b