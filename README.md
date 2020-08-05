## Disclaimer

Your warranty is now void. Proceed at your own risk.

## Requirements

You must have the latest Windows Updates from ASUS. Check your `C:\Windows\System32\ASUSACCI` folder:
```powershell
PS C:\Windows\System32\ASUSACCI> dir


    Directory: C:\Windows\System32\ASUSACCI


Mode                 LastWriteTime         Length Name
----                 -------------         ------ ----
-a----         7/22/2020   7:52 PM         889240 ArmouryCrateControlInterface.exe
-a----         7/22/2020   7:52 PM         344480 ArmouryCrateKeyControl.exe
-a----         7/22/2020   7:52 PM        3764632 cpprest141_2_10.dll
```

Recommended: Rename `ArmouryCrateKeyControl.exe` into something else (e.g. `ArmouryCrateKeyControl_original.exe`) so Asus' software won't interfere with this program.

## Change the Behavior

By default, it will launch Task Manager when you press the ROG Key. Change `commandWithArgs` in `main.go` to your desired program. You can compile your `.ahk` to `.exe` and run your macros. Then build the binary again (see below).

## How to Build

1. Install golang 1.14+ if you don't have it already
2. Install `rsrc`: `go get github.com/akavel/rsrc`
3. Generate `syso` file: `\path\to\rsrc.exe -manifest ROGKeyRebind.exe.manifest -o ROGKeyRebind.exe.syso`
4. Build the binary: `go build -ldflags -H=windowsgui github.com/zllovesuki/ROGKeyRebind`

Recommend running ROGKeyRebind.exe on startup in Task Scheduler.

## Developing

Remove the `-ldflags -H=windowsgui` when you run or build, then you will see the console