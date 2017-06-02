#!/bin/bash

pushd src

APP_DIR=TestApp.AppDir1

./linuxdeploy -exe ../tests/TestApp/TestApp -appdir $APP_DIR -strip -overwrite -libs ../tests/TestLib/ -qmldir ../tests/TestApp/ -log linuxdeploy.log

rc=$?; if [[ $rc != 0 ]]; then exit 1; fi

until xset -q
do
  echo "Waiting for X server to start..."
  sleep 1;
done

echo "Deployed all needed libraries. Trying to launch..."

pushd $APP_DIR

unset QT_PLUGIN_PATH
unset LD_LIBRARY_PATH
unset QTDIR

export QML_IMPORT_TRACE=1
export QT_DEBUG_PLUGINS=1

LD_DEBUG=libs ./TestApp
rc=$?

echo "Application finished with code $rc."

popd # appdir

popd # src

if [[ $rc != 0 ]]; then exit 1; fi
