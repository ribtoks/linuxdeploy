package main

import (
  "log"
  "strings"
  "os/exec"
  "path/filepath"
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
  qtEnv map[QMakeKey]string
}

func (qd *QtDeployer) queryQtEnv() error {
  out, err := exec.Command("qmake", "-query").Output()
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

func (qd *QtDeployer) PluginsPath() string {
  return qd.qtEnv[QT_INSTALL_PLUGINS]
}

func (ad *AppDeployer) processQtLibs() {
  for request := range ad.qtChannel {
    libname := strings.ToLower(request.Basename())

    if !strings.HasPrefix(libname, "libqt") { continue }

    if strings.HasPrefix(libname, "libqt5gui") {
      ad.deployQtPlugin("platforms/libqxcb.so")
    } else if strings.HasPrefix(libname, "libqt5svg") {
      ad.deployQtPlugin("iconengines/libqsvgicon.so")
    } else if strings.HasPrefix(libname, "libqt5printsupport") {
      ad.deployQtPlugin("printsupport/libcupsprintersupport.so")
    } else if strings.HasPrefix(libname, "libqt5opengl") ||
      strings.HasPrefix(libname, "libqt5xcbqpa") {
      ad.deployRecursively(filepath.Join(ad.qtDeployer.PluginsPath(), "xcbglintegrations"), "plugins")
    } else if strings.HasPrefix(libname, "libqt5network") {
      ad.deployRecursively(filepath.Join(ad.qtDeployer.PluginsPath(), "bearer"), "plugins")
    }

    ad.waitGroup.Done()
  }
}

func (ad *AppDeployer) deployQtPlugin(relpath string) {
  go func() {
    ad.libsChannel <- DeployRequest {
      sourcePath: relpath,
      sourceRoot: ad.qtDeployer.PluginsPath(),
      targetRoot: ".",
      isLddDependency: true,
    }
  }()
}
