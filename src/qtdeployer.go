/*
 * This file is a part of linuxdeploy - tool for
 * creating standalone applications for Linux
 *
 * Copyright (C) 2017 Taras Kushnir <kushnirTV@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the MIT License.

 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 */

package main

import (
  "log"
  "strings"
  "os"
  "os/exec"
  "io/ioutil"
  "errors"
  "path/filepath"
  "encoding/json"
)

type QMakeKey int

const (
  QT_INSTALL_PREFIX QMakeKey = iota
  QT_INSTALL_ARCHDATA
  QT_INSTALL_DATA
  QT_INSTALL_DOCS
  QT_INSTALL_HEADERS
  QT_INSTALL_LIBS
  QT_INSTALL_LIBEXECS
  QT_INSTALL_BINS
  QT_INSTALL_TESTS
  QT_INSTALL_PLUGINS
  QT_INSTALL_IMPORTS
  QT_INSTALL_QML
  QT_INSTALL_TRANSLATIONS
  QT_INSTALL_CONFIGURATION
  QT_INSTALL_EXAMPLES
  QT_INSTALL_DEMOS
  QT_HOST_PREFIX
  QT_HOST_DATA
  QT_HOST_BINS
  QT_HOST_LIBS
  QMAKE_VERSION
  QT_VERSION
)

type QtDeployer struct {
  qmakePath string
  qmakeVars map[string]string
  deployedQmlImports map[string]bool
  qtEnv map[QMakeKey]string
  qmlImportDirs []string
  privateWidgetsDeployed bool
  qtEnvironmentSet bool
}

func (qd *QtDeployer) queryQtEnv() error {
  log.Printf("Querying qmake environment using %v", qd.qmakePath)
  if len(qd.qmakePath) == 0 { return errors.New("QMake has not been resolved") }

  out, err := exec.Command(qd.qmakePath, "-query").Output()
  if err != nil { return err }

  output := string(out)
  // TODO: probably switch to bytes.Split for better performance
  lines := strings.Split(output, "\n")

  for _, line := range lines {
    line = strings.TrimSpace(line)
    if len(line) == 0 { continue }
    parts := strings.Split(line, ":")

    if len(parts) != 2 {
      log.Printf("Unexpected qmake output: %v", line)
      continue
    }

    qd.qmakeVars[parts[0]] = parts[1]
  }

  qd.parseQtVars()
  log.Println("Parsed qmake output: %v", qd.qtEnv)
  qd.qtEnvironmentSet = true
  return nil
}

func (qd *QtDeployer) parseQtVars() {
  qd.qtEnv[QT_INSTALL_PREFIX], _ = qd.qmakeVars["QT_INSTALL_PREFIX"]
  qd.qtEnv[QT_INSTALL_ARCHDATA], _ = qd.qmakeVars["QT_INSTALL_ARCHDATA"]
  qd.qtEnv[QT_INSTALL_DATA], _ = qd.qmakeVars["QT_INSTALL_DATA"]
  qd.qtEnv[QT_INSTALL_DOCS], _ = qd.qmakeVars["QT_INSTALL_DOCS"]
  qd.qtEnv[QT_INSTALL_HEADERS], _ = qd.qmakeVars["QT_INSTALL_HEADERS"]
  qd.qtEnv[QT_INSTALL_LIBS], _ = qd.qmakeVars["QT_INSTALL_LIBS"]
  qd.qtEnv[QT_INSTALL_LIBEXECS], _ = qd.qmakeVars["QT_INSTALL_LIBEXECS"]
  qd.qtEnv[QT_INSTALL_BINS], _ = qd.qmakeVars["QT_INSTALL_BINS"]
  qd.qtEnv[QT_INSTALL_PLUGINS], _ = qd.qmakeVars["QT_INSTALL_PLUGINS"]
  qd.qtEnv[QT_INSTALL_IMPORTS], _ = qd.qmakeVars["QT_INSTALL_IMPORTS"]
  qd.qtEnv[QT_INSTALL_QML], _ = qd.qmakeVars["QT_INSTALL_QML"]
  qd.qtEnv[QT_INSTALL_TRANSLATIONS], _ = qd.qmakeVars["QT_INSTALL_TRANSLATIONS"]
  qd.qtEnv[QT_INSTALL_CONFIGURATION], _ = qd.qmakeVars["QT_INSTALL_CONFIGURATION"]
  qd.qtEnv[QT_HOST_PREFIX], _ = qd.qmakeVars["QT_HOST_PREFIX"]
  qd.qtEnv[QT_HOST_DATA], _ = qd.qmakeVars["QT_HOST_DATA"]
  qd.qtEnv[QT_HOST_BINS], _ = qd.qmakeVars["QT_HOST_BINS"]
  qd.qtEnv[QT_HOST_LIBS], _ = qd.qmakeVars["QT_HOST_LIBS"]
  qd.qtEnv[QMAKE_VERSION], _ = qd.qmakeVars["QMAKE_VERSION"]
  qd.qtEnv[QT_VERSION], _ = qd.qmakeVars["QT_VERSION"]
}

func (qd *QtDeployer) BinPath() string {
  return qd.qtEnv[QT_INSTALL_BINS]
}

func (qd *QtDeployer) PluginsPath() string {
  return qd.qtEnv[QT_INSTALL_PLUGINS]
}

func (qd *QtDeployer) LibExecsPath() string {
  return qd.qtEnv[QT_INSTALL_LIBEXECS]
}

func (qd *QtDeployer) DataPath() string {
  return qd.qtEnv[QT_INSTALL_DATA]
}

func (qd *QtDeployer) TranslationsPath() string {
  return qd.qtEnv[QT_INSTALL_TRANSLATIONS]
}

func (qd *QtDeployer) QmlPath() string {
  return qd.qtEnv[QT_INSTALL_QML]
}

func (qd *QtDeployer) accountQmlImport(path string) {
  qd.deployedQmlImports[path] = true
}

func (qd *QtDeployer) isQmlImportDeployed(path string) (deployed bool) {
  // TODO: also check directory hierarchy?
  _, deployed = qd.deployedQmlImports[path]
  return deployed
}

func (ad *AppDeployer) processQtLibTasks() {
  if !ad.qtDeployer.qtEnvironmentSet {
    log.Printf("Qt Environment is not initialized")
    return
  }

  go ad.deployQmlImports()

  for libraryPath := range ad.qtChannel {
    ad.processQtLibTask(libraryPath)
    // rpath should be changed for all qt libs
    ad.addFixRPathTask(libraryPath)

    ad.waitGroup.Done()
  }

  log.Printf("Qt libraries processing finished")
}

func (ad *AppDeployer) processQtLibTask(libraryPath string) {
  libraryBasename := filepath.Base(libraryPath)
  libname := strings.ToLower(libraryBasename)

  if !strings.HasPrefix(libname, "libqt") { log.Fatal("Can only accept Qt libraries") }
  log.Printf("Inspecting Qt lib: %v", libraryBasename)

  deployFlags := LDD_DEPENDENCY_FLAG | DEPLOY_ONLY_LIBRARIES_FLAG | FIX_RPATH_FLAG

  if strings.HasPrefix(libname, "libqt5gui") {
    ad.addQtPluginTask("platforms/libqxcb.so")
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "imageformats", "plugins", deployFlags)
  } else
  if strings.HasPrefix(libname, "libqt5svg") {
    ad.addQtPluginTask("iconengines/libqsvgicon.so")
  } else
  if strings.HasPrefix(libname, "libqt5printsupport") {
    ad.addQtPluginTask("printsupport/libcupsprintersupport.so")
  } else
  if strings.HasPrefix(libname, "libqt5opengl") ||
    strings.HasPrefix(libname, "libqt5xcbqpa") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "xcbglintegrations", "plugins", deployFlags)
  } else
  if strings.HasPrefix(libname, "libqt5network") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "bearer", "plugins", deployFlags)
  } else
  if strings.HasPrefix(libname, "libqt5sql") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "sqldrivers", "plugins", deployFlags)
  } else
  if strings.HasPrefix(libname, "libqt5multimedia") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "mediaservice", "plugins", deployFlags)
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "audio", "plugins", deployFlags)
  } else
  if strings.HasPrefix(libname, "libqt5webenginecore") {
    ad.addCopyQtDepTask(ad.qtDeployer.LibExecsPath(), "QtWebEngineProcess", "libexecs")
    ad.copyRecursively(ad.qtDeployer.DataPath(), "resources", ".")
    ad.copyRecursively(ad.qtDeployer.TranslationsPath(), "qtwebengine_locales", "translations")
  } else
  if strings.HasPrefix(libname, "libqt5core") {
    go ad.patchQtCore(libraryPath)
  }
}

// copies one file
func (ad *AppDeployer) addCopyQtDepTask(sourceRoot, sourcePath, targetPath string) error {
  path := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Copy once %v into %v", path, targetPath)
  relativePath, err := filepath.Rel(sourceRoot, path)
  if err != nil {
    log.Println(err)
  }

  ad.waitGroup.Add(1)
  go func() {
    ad.copyChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: relativePath,
      targetPath: targetPath,
      flags: FIX_RPATH_FLAG,
    }
  }()

  return err
}

func (ad *AppDeployer) addQtPluginTask(relpath string) {
  log.Printf("Deploying additional Qt plugin: %v", relpath)
  ad.addLibTask(ad.qtDeployer.PluginsPath(), relpath, "plugins", FIX_RPATH_FLAG)
}

func (ad *AppDeployer) deployQmlImports() error {
  // rescue agains premature finish of the main loop
  ad.waitGroup.Add(1)
  defer ad.waitGroup.Done()

  log.Printf("Processing QML imports from %v", ad.qtDeployer.qmlImportDirs)

  scannerPath := filepath.Join(ad.qtDeployer.BinPath(), "qmlimportscanner")

  if _, err := os.Stat(scannerPath); err != nil {
    if scannerPath, err = exec.LookPath("qmlimportscanner"); err != nil {
      log.Printf("Cannot find qmlimportscanner")
      return err
    }
  }

  log.Printf("QML import scanner: %v", scannerPath)

  args := make([]string, 0, 10)
  for _, qmldir := range ad.qtDeployer.qmlImportDirs {
    args = append(args, "-rootPath")
    args = append(args, qmldir)
  }

  args = append(args, "-importPath")
  args = append(args, ad.qtDeployer.QmlPath())

  out, err := exec.Command(scannerPath, args...).Output()
  if err != nil {
    log.Printf("QML import scanner failed with %v", err)
    return err
  }

  err = ad.processQmlImportsJson(out)
  return err
}

type QmlImport struct {
  Classname string `json:"classname,omitempty"`
  Name string `json:"name,omitempty"`
  Path string `json:"path,omitempty"`
  Plugin string `json:"plugin,omitempty"`
  ImportType string `json:"type,omitempty"`
  Version string `json:"version,omitempty"`
}

func (ad *AppDeployer) processQmlImportsJson(jsonRaw []byte) (err error) {
  log.Printf("Parsing QML imports")

  var qmlImports []QmlImport
  err = json.Unmarshal(jsonRaw, &qmlImports)
  if err != nil { return err }
  log.Printf("Parsed %v imports", len(qmlImports))

  sourceRoot := ad.qtDeployer.QmlPath()

  for _, qmlImport := range qmlImports {
    relativePath, err := filepath.Rel(sourceRoot, qmlImport.Path)

    if err != nil || len(qmlImport.Name) == 0 {
      log.Printf("Skipping import %v", qmlImport)
      continue
    }

    if qmlImport.ImportType != "module" {
      log.Printf("Skipping non-module import %v", qmlImport)
      continue
    }

    if len(qmlImport.Path) == 0 {
      log.Printf("Skipping import without path %v", qmlImport)
      continue
    }

    if ad.qtDeployer.isQmlImportDeployed(qmlImport.Path) {
      log.Printf("Skipping already deployed QML import %v", qmlImport.Path)
      continue
    }

    if (qmlImport.Name == "QtQuick.Controls") && !ad.qtDeployer.privateWidgetsDeployed {
      ad.qtDeployer.privateWidgetsDeployed = true
      log.Printf("Deploying private widgets for QtQuick.Controls")
      ad.deployRecursively(sourceRoot, "QtQuick/PrivateWidgets", "qml", FIX_RPATH_FLAG)
    }

    log.Printf("Deploying QML import %v", qmlImport.Path)
    ad.qtDeployer.accountQmlImport(qmlImport.Path)
    ad.deployRecursively(sourceRoot, relativePath, "qml", FIX_RPATH_FLAG)
  }

  return nil
}

func (ad *AppDeployer) patchQtCore(libraryPath string) {
  // rescue agains premature finish of the main loop
  ad.waitGroup.Add(1)
  defer ad.waitGroup.Done()

  log.Printf("Patching libQt5Core at path %v", libraryPath)
  err := patchQtCore(libraryPath)
  if err != nil {
    log.Printf("QtCore patching failed! %v", err)
  }
}

func patchQtCore(path string) error {
  fi, err := os.Stat(path)
  if err != nil { return err }

  originalMode := fi.Mode()

  contents, err := ioutil.ReadFile(path)
  if err != nil { return err }

  // this list originates from https://github.com/probonopd/linuxdeployqt
  replaceVariable(contents, "qt_prfxpath=", ".");
  replaceVariable(contents, "qt_adatpath=", ".");
  replaceVariable(contents, "qt_docspath=", "doc");
  replaceVariable(contents, "qt_hdrspath=", "include");
  replaceVariable(contents, "qt_libspath=", "lib");
  replaceVariable(contents, "qt_lbexpath=", "libexec");
  replaceVariable(contents, "qt_binspath=", "bin");
  replaceVariable(contents, "qt_plugpath=", "plugins");
  replaceVariable(contents, "qt_impspath=", "imports");
  replaceVariable(contents, "qt_qml2path=", "qml");
  replaceVariable(contents, "qt_datapath=", ".");
  replaceVariable(contents, "qt_trnspath=", "translations");
  replaceVariable(contents, "qt_xmplpath=", "examples");
  replaceVariable(contents, "qt_demopath=", "demos");
  replaceVariable(contents, "qt_tstspath=", "tests");
  replaceVariable(contents, "qt_hpfxpath=", ".");
  replaceVariable(contents, "qt_hbinpath=", "bin");
  replaceVariable(contents, "qt_hdatpath=", ".");
  replaceVariable(contents, "qt_stngpath=", "."); // e.g., /opt/qt53/etc/xdg; does it load Trolltech.conf from there?

  /* Qt on Arch Linux comes with more hardcoded paths
  * https://github.com/probonopd/linuxdeployqt/issues/98
  replaceVariable(contents, "lib/qt/libexec", "libexec");
  replaceVariable(contents, "lib/qt/plugins", "plugins");
  replaceVariable(contents, "lib/qt/imports", "imports");
  replaceVariable(contents, "lib/qt/qml", "qml");
  replaceVariable(contents, "lib/qt", "");
  replaceVariable(contents, "share/doc/qt", "doc");
  replaceVariable(contents, "include/qt", "include");
  replaceVariable(contents, "share/qt", "");
  replaceVariable(contents, "share/qt/translations", "translations");
  replaceVariable(contents, "share/doc/qt/examples", "examples");
  */

  err = ioutil.WriteFile(path, contents, originalMode)
  return err
}
