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
  "path/filepath"
  "fmt"
  "os"
  "os/exec"
  "sync"
)

// list of modules from windeployqt
const (
  QtBluetoothModule int64 = 1 << iota
  QtCLuceneModule
  QtConcurrentModule
  QtCoreModule
  QtDeclarativeModule
  QtDesignerComponents
  QtDesignerModule
  QtGuiModule
  QtCluceneModule
  QtHelpModule
  QtMultimediaModule
  QtMultimediaWidgetsModule
  QtMultimediaQuickModule
  QtNetworkModule
  QtNfcModule
  QtOpenGLModule
  QtPositioningModule
  QtPrintSupportModule
  QtQmlModule
  QtQuickModule
  QtQuickParticlesModule
  QtScriptModule
  QtScriptToolsModule
  QtSensorsModule
  QtSerialPortModule
  QtSqlModule
  QtSvgModule
  QtTestModule
  QtWidgetsModule
  _ // QtWinExtrasModule
  QtXmlModule
  QtXmlPatternsModule
  QtWebKitModule
  QtWebKitWidgetsModule
  QtQuickWidgetsModule
  QtWebSocketsModule
  QtEnginioModule
  QtWebEngineCoreModule
  QtWebEngineModule
  QtWebEngineWidgetsModule
  QtQmlToolingModule
  Qt3DCoreModule
  Qt3DRendererModule
  Qt3DQuickModule
  Qt3DQuickRendererModule
  Qt3DInputModule
  QtLocationModule
  QtWebChannelModule
  QtTextToSpeechModule
  QtSerialBusModule
)

var moduleToTranslationMap = map[string]string {
  /*QtBluetoothModule:*/ "qt5bluetooth": "",
  /*QtCLuceneModule:*/"qt5clucene": "qt_help",
  /*QtConcurrentModule:*/ "qt5concurrent": "qtbase",
  /*QtCoreModule:*/ "qt5core": "qtbase",
  /*QtDeclarativeModule:*/ "qt5declarative": "qtquick1",
  /*QtDesignerComponents:*/ "qt5designercomponents": "",
  /*QtDesignerModule:*/ "qt5designer": "",
  /*QtGuiModule:*/ "qt5gui": "qtbase",
  /*QtHelpModule:*/ "qt5help": "qt_help",
  /*QtMultimediaModule:*/ "qt5multimedia": "qtmultimedia",
  /*QtMultimediaWidgetsModule:*/ "qt5multimediawidgets": "qtmultimedia",
  /*QtMultimediaQuickModule:*/ "qt5multimediaquick_p": "qtmultimedia",
  /*QtNetworkModule:*/ "qt5network": "qtbase",
  /*QtNfcModule:*/ "qt5nfc": "",
  /*QtOpenGLModule:*/ "qt5opengl": "",
  /*QtPositioningModule:*/ "qt5positioning": "",
  /*QtPrintSupportModule:*/ "qt5printsupport": "",
  /*QtQmlModule:*/ "qt5qml": "qtdeclarative",
  /*QtQuickModule:*/ "qt5quick": "qtdeclarative",
  /*QtQuickParticlesModule:*/ "qt5quickparticles": "",
  /*QtScriptModule:*/ "qt5script": "qtscript",
  /*QtScriptToolsModule:*/ "qt5scripttools": "qtscript",
  /*QtSensorsModule:*/ "qt5sensors": "",
  /*QtSerialPortModule:*/ "qt5serialport": "qtserialport",
  /*QtSqlModule:*/ "qt5sql": "qtbase",
  /*QtSvgModule:*/ "qt5svg": "",
  /*QtTestModule:*/ "qt5test": "",
  /*QtWidgetsModule:*/ "qt5widgets": "qtbase",
  // QtWinExtrasModule:*/ "qt5winextras": "",
  /*QtXmlModule:*/ "qt5xml": "qtbase",
  /*QtXmlPatternsModule:*/ "qt5xmlpatterns": "qtxmlpatterns",
  /*QtWebKitModule:*/ "qt5webkit": "qtwebengine",
  /*QtWebKitWidgetsModule:*/ "qt5webkitwidgets": "",
  /*QtQuickWidgetsModule:*/ "qt5quickwidgets": "",
  /*QtWebSocketsModule:*/ "qt5websockets": "qtwebsockets",
  /*QtEnginioModule:*/ "enginio": "",
  /*QtWebEngineCoreModule:*/ "qt5webenginecore": "",
  /*QtWebEngineModule:*/ "qt5webengine": "",
  /*QtWebEngineWidgetsModule:*/ "qt5webenginewidgets": "",
  /*QtQmlToolingModule:*/ "qt5qmltooling": "qmltooling",
  /*Qt3DCoreModule:*/ "qt53dcore": "",
  /*Qt3DRendererModule:*/ "qt53drenderer": "",
  /*Qt3DQuickModule:*/ "qt53dquick": "",
  /*Qt3DQuickRendererModule:*/ "qt53dquickrenderer": "",
  /*Qt3DInputModule:*/ "qt53dinput": "",
  /*QtLocationModule:*/ "qt5location": "",
  /*QtWebChannelModule:*/ "qt5webchannel": "",
  /*QtTextToSpeechModule:*/ "qt5texttospeech": "",
  /*QtSerialBusModule:*/ "qt5serialbus": "",
}

func (qd *QtDeployer) accountQtLibrary(libname string) {
  extensionIndex := strings.LastIndex(libname, ".so")
  if extensionIndex == -1 { return }

  libprefix := libname[3:extensionIndex]
  if translation, ok := moduleToTranslationMap[libprefix]; ok {
    if len(translation) > 0 {
      qd.translationsRequired[translation] = true
      log.Printf("Accounted translation %v for lib %v", translation, libname)
    }
  } else {
    log.Printf("Translations unknown for module: %v", libname)
  }
}

func (ad *AppDeployer) deployQtTranslations(translationsRoot string, mainWaitGroup *sync.WaitGroup) {
  defer mainWaitGroup.Done()
  if !ad.qtDeployer.qtEnvironmentSet { return }

  qtTranslationsPath := ad.qtDeployer.TranslationsPath()

  languages := retrieveAvailableLanguages(qtTranslationsPath)
  if len(languages) == 0 { return }

  lconvertPath := filepath.Join(ad.qtDeployer.BinPath(), "lconvert")

  if _, err := os.Stat(lconvertPath); err != nil {
    if lconvertPath, err = exec.LookPath("lconvert"); err != nil {
      log.Printf("Cannot find lconvert")
      return
    }
  }

  log.Printf("Required translations: %v", ad.qtDeployer.translationsRequired)
  ensureDirExists(filepath.Join(translationsRoot, "dummyfile"))

  var wg sync.WaitGroup

  for _, lang := range languages {
    wg.Add(1)
    go ad.deployLanguage(lang, lconvertPath, translationsRoot, &wg)
  }

  wg.Wait()
  log.Printf("Translations generations finished")
}

func (ad *AppDeployer) deployLanguage(lang, lconvertPath, translationsRoot string, wg *sync.WaitGroup) {
  defer wg.Done()

  qtTranslationsPath := ad.qtDeployer.TranslationsPath()

  arguments := make([]string, 0, 10)
  // generate combined translation files for each language
  outputFile := fmt.Sprintf("qt_%s.qm", lang)
  outputFilepath := filepath.Join(translationsRoot, outputFile)

  arguments = append(arguments, "-o", outputFilepath)

  for module, _ := range ad.qtDeployer.translationsRequired {
    trFile := fmt.Sprintf("%s_%s.qm", module, lang)
    trFilepath := filepath.Join(qtTranslationsPath, trFile)
    arguments = append(arguments, trFilepath)
  }

  log.Printf("Launching lconvert with arguments %v", arguments)

  err := exec.Command(lconvertPath, arguments...).Run()
  if err != nil {
    log.Printf("lconvert failed with %v", err)
  } else {
    log.Printf("Generated translations file %v", outputFile)
  }
}

func retrieveAvailableLanguages(translationsRoot string) []string {
  log.Printf("Translations: checking available languages in %v", translationsRoot)

  languages := make([]string, 0, 10)

  err := filepath.Walk(translationsRoot, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    basename := strings.ToLower(filepath.Base(path))

    if !strings.HasPrefix(basename, "qtbase_") || !strings.HasSuffix(basename, ".qm") {
      return nil
    }

    // language between qtbase_ and .qm
    lastIndex := len(basename) - 3
    lang := basename[7:lastIndex]
    languages = append(languages, lang)

    return nil
  })

  if err != nil { log.Printf("Error while searching translations: %v", err) }
  log.Printf("Found qt translations languages %v", languages)

  return languages
}
