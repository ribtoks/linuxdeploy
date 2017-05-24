/*
 * This file is a part of linuxdeploy - tool for
 * creating standalone applications for Linux
 * Copyright (C) 2017 Taras Kushnir <kushnirTV@gmail.com>
 *
 * linuxdeploy is distributed under the GNU General Public License, version 3.0
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
  "log"
  "strings"
  "os"
  "os/exec"
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
  qmlImportsDeployed bool
  qtEnvironmentSet bool
}

func (qd *QtDeployer) queryQtEnv() error {
  log.Printf("Querying qmake environment using %v", qd.qmakePath)
  if len(qd.qmakePath) == 0 { return errors.New("QMake has not been resolved") }

  out, err := exec.Command(qd.qmakePath, "-query").Output()
  if err != nil { return err }

  output := string(out)
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

func (ad *AppDeployer) processQtLibs() {
  if !ad.qtDeployer.qtEnvironmentSet {
    log.Printf("Qt Environment is not initialized")
    return
  }

  if (!ad.qtDeployer.qmlImportsDeployed) {
    ad.qtDeployer.qmlImportsDeployed = true
    go ad.deployQmlImports()
  }

  for request := range ad.qtChannel {
    ad.processQtLibrary(request)
    ad.waitGroup.Done()
  }
}

func (ad *AppDeployer) processQtLibrary(request *DeployRequest) {
  libname := strings.ToLower(request.Basename())

  if !strings.HasPrefix(libname, "libqt") {
    return
  }

  log.Printf("Inspecting Qt lib: %v", request.Basename())

  if strings.HasPrefix(libname, "libqt5gui") {
    ad.deployQtPlugin("platforms/libqxcb.so")
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "imageformats", "plugins", DEPLOY_LIBRARIES)
  } else
  if strings.HasPrefix(libname, "libqt5svg") {
    ad.deployQtPlugin("iconengines/libqsvgicon.so")
  } else
  if strings.HasPrefix(libname, "libqt5printsupport") {
    ad.deployQtPlugin("printsupport/libcupsprintersupport.so")
  } else
  if strings.HasPrefix(libname, "libqt5opengl") ||
    strings.HasPrefix(libname, "libqt5xcbqpa") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "xcbglintegrations", "plugins", DEPLOY_LIBRARIES)
  } else
  if strings.HasPrefix(libname, "libqt5network") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "bearer", "plugins", DEPLOY_LIBRARIES)
  } else
  if strings.HasPrefix(libname, "libqt5sql") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "sqldrivers", "plugins", DEPLOY_LIBRARIES)
  } else
  if strings.HasPrefix(libname, "libqt5multimedia") {
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "mediaservice", "plugins", DEPLOY_LIBRARIES)
    ad.deployRecursively(ad.qtDeployer.PluginsPath(), "audio", "plugins", DEPLOY_LIBRARIES)
  } else
  if strings.HasPrefix(libname, "libqt5webenginecore") {
    ad.copyOnce(ad.qtDeployer.LibExecsPath(), "QtWebEngineProcess", "libexecs")
    ad.copyRecursively(ad.qtDeployer.DataPath(), "resources", ".")
    ad.copyRecursively(ad.qtDeployer.TranslationsPath(), "qtwebengine_locales", "translations")
  }
}

func (ad *AppDeployer) deployQtPlugin(relpath string) {
  log.Printf("Deploying additional Qt plugin: %v", relpath)
  ad.waitGroup.Add(1)
  go func() {
    ad.libsChannel <- &DeployRequest {
      sourcePath: relpath,
      sourceRoot: ad.qtDeployer.PluginsPath(),
      targetRoot: "plugins",
      isLddDependency: true,
    }
  }()
}

func (ad *AppDeployer) deployQmlImports() error {
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

    if qmlImport.Name == "QtQuick.Controls" {
      log.Printf("Deploying private widgets for QtQuick.Controls")
      ad.deployRecursively(sourceRoot, "QtQuick/PrivateWidgets", "qml", DEPLOY_EVERYTHING)
    }

    log.Printf("Deploying QML import %v", qmlImport.Path)
    ad.qtDeployer.accountQmlImport(qmlImport.Path)
    ad.deployRecursively(sourceRoot, relativePath, "qml", DEPLOY_EVERYTHING)
  }

  return nil
}
