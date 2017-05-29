#!/bin/bash

pushd src

APP_DIR=TestApp.AppDir

./linuxdeploy -exe ../tests/TestApp/TestApp -appdir $APP_DIR -overwrite -libs ../tests/TestLib/ -qmldir ../tests/TestApp/

rc=$?; if [[ $rc != 0 ]]; then exit 1; fi

pushd $APP_DIR

unset LD_LIBRARY_PATH
export LD_LIBRARY_PATH=""

LD_DEBUG=libs ./TestApp
