#-------------------------------------------------
#
# Project created by QtCreator 2017-05-29T20:07:50
#
#-------------------------------------------------

QT       -= core gui

TARGET = testlib
TEMPLATE = lib

DEFINES += TESTLIB_LIBRARY

SOURCES += testlib.cpp

HEADERS += testlib.h\
        testlib_global.h

unix {
    target.path = /usr/lib
    INSTALLS += target
}
