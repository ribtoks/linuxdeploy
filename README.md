# linuxdeploy
A tool for deploying standalone Linux applications

[![Build Status](https://travis-ci.org/Ribtoks/linuxdeploy.svg?branch=master)](https://travis-ci.org/Ribtoks/linuxdeploy)

# Description
**linuxdeploy** inspects the executable file and deploys it alongside with all the dependencies to a specified location. Afterwards RPATH is fixed correctly so the deployed executable only uses deployed libraries. Main use-case for this tool is _deploying Qt applications on Linux without pain in the format of AppImage_, however your mileage may vary.

Find more developers documentation in the [CONTRIBUTING.md](https://github.com/Ribtoks/linuxdeploy/blob/master/CONTRIBUTING.md)

# Build

As easy as:

    cd src
    go build -o linuxdeploy
    
# Dependencies

You have to have in your `PATH`:

* `ldd` (checking dso dependencies)
* [patchelf](https://anonscm.debian.org/cgit/collab-maint/patchelf.git/) (patching `RPATH` in binaries)
* `strip` (optionally to remove debug symbols from binaries)
 
# Usage

## Simple usage

Most simple usage of this tool:

    linuxdeploy -exe /path/to/myexe -appdir myexe.AppDir -icon /path/to/icon 
        -gen-desktop -default-blacklist -out appimage
        
    appimagetool --verbose -n myexe.AppDir "myexe.AppImage"
   
These commands will deploy application `myexe` and it's dependencies to the directory `./myexe.AppDir/` packing in the AppImage-compatible structure. Afterwards AppImage is generated with an [AppImageTool](https://github.com/probonopd/AppImageKit).

## Deploying Qt

**linuxdeploy** is capable of deploying all Qt's dependencies of your app: libraries, private widgets, QML imports and translations. Optionally you can specify path to the `qmake` executable and **linuxdeploy** will derive Qt Environment from it. You can specify additional directories to search for qml imports using a repeatable `-qmldir` switch.

## Other features

Usually when creating AppImage you don't need to deploy _all_ the libraries (like _libstdc++_ or _libdbus_). **linuxdeploy** supports ignore list as a command-line parameter `-blacklist`. It is path to a file with an ignore per line where ignore is a prefix of the library to skip (e.g. if you need to ignore _libstdc++.so.6_ you can have a line _libstdc++_ in the blacklist file). Also you have a default blacklist which can be checked out in the `src/blacklist.go` file and can be added with `-default-blacklist` cmdline switch.

**linuxdeploy** can also generate a desktop file in the deployment directory. Also it will fill-in information about icon and AppRun link in case you're deploying AppImage.

Every binary deployed (original exe and dependent libs) can be stripped if you specify cmdline switch `-strip`.

## Command line switches:
 
    -exe string
     	Path to the executable to deploy
    -appdir string
     	Path to the destination deployment directory or AppDir (if 'type' is appimage)
    -libs value
     	Additional libraries search paths (repeatable)
    -qmake string
     	Path to qmake
    -qmldir value
     	Additional QML imports dir (repeatable)
    -blacklist string
     	Path to the additional libraries blacklist file (default "libs.blacklist")
    -default-blacklist
     	Add default blacklist
    -gen-desktop
     	Generate desktop file
    -icon string
     	Path the exe's icon (used for desktop file)
    -log string
     	Path to the logfile (default "linuxdeploy.log")
    -out string
     	Type of the generated output (default "appimage")
    -overwrite
     	Overwrite output if preset
    -stdout
     	Log to stdout and to logfile
    -strip
     	Run strip on binaries
        
# Known issues

The only working `patchelf` right now is from the [Debian's repository](https://anonscm.debian.org/cgit/collab-maint/patchelf.git/). [Vanilla patchelf](https://github.com/NixOS/patchelf) damages `libQt5Core.so` library.

# Disclaimer

I wrote this tool because [linuxdeployqt](https://github.com/probonopd/linuxdeployqt/) ~was too buggy for me~ did not work well for me. Now this implementation successfully deploys [more or less complex desktop Qt/Qml app](https://github.com/ribtoks/xpiks) and works a lot faster then the former.

Pull Requests and feedback are more than welcome! Please check out [CONTRIBUTING.md](https://github.com/Ribtoks/linuxdeploy/blob/master/CONTRIBUTING.md) for more details and developers documentation on the internals.
