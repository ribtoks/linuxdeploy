# linuxdeploy
Tool for creating standalone Linux applications

[![Build Status](https://travis-ci.org/Ribtoks/linuxdeploy.svg?branch=master)](https://travis-ci.org/Ribtoks/linuxdeploy)
 
 Command line switches:
 
    -appdir string
     	Path to the AppDir (if 'type' is appimage)
    -blacklist string
     	Path to the additional libraries blacklist file (default "libs.blacklist")
    -default-blacklist
     	Add default blacklist
    -exe string
     	Path to the executable to deploy
    -gen-desktop
     	Generate desktop file
    -icon string
     	Path the exe's icon (used for desktop file)
    -libs value
     	Additional libraries search paths (repeatable)
    -log string
     	Path to the logfile (default "linuxdeploy.log")
    -out string
     	Type of the generated output (default "appimage")
    -overwrite
     	Overwrite output if preset
    -qmake string
     	Path to qmake
    -qmldir value
     	Additional QML imports dir (repeatable)
    -stdout
     	Log to stdout and to logfile
    -strip
     	Run strip on binaries
