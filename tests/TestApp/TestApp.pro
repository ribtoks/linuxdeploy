TEMPLATE = app

QT += qml quick svg
CONFIG += c++11

INCLUDEPATH += $$PWD/../TestLib/
LIBS += -L$$PWD/../TestLib/build-Debug/
LIBS += -L$$PWD/../TestLib/
LIBS += -ltestlib

SOURCES += main.cpp
RESOURCES += qml.qrc

# Additional import path used to resolve QML modules in Qt Creator's code model
QML_IMPORT_PATH =

# Default rules for deployment.
include(deployment.pri)
