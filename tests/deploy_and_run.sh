#!/bin/bash

pushd src

APP_DIR=TestApp.AppDir

./linuxdeploy -exe ../tests/TestApp/TestApp -appdir $APP_DIR -overwrite -libs ../tests/TestLib/ -qmldir ../tests/TestApp/

rc=$?; if [[ $rc != 0 ]]; then exit 1; fi

echo "Deployed all needed libraries. Trying to launch..."

pushd $APP_DIR

unset QT_PLUGIN_PATH
unset LD_LIBRARY_PATH
unset QTDIR

LD_DEBUG=libs ./TestApp

popd # appdir

popd # src
