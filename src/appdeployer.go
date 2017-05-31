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
  "os"
  "strings"
  "sync"
  "path/filepath"
  "time"
)

const (
  LDD_DEPENDENCY_FLAG Bitmask = 1 << iota // check ldd deps for given item
  FIX_RPATH_FLAG // fix rpath for qt-related libs/plugins
  DEPLOY_ONLY_LIBRARIES_FLAG
)

const (
  LDD_AND_RPATH_FLAG = LDD_DEPENDENCY_FLAG | FIX_RPATH_FLAG
  LIBRARIES_AND_RPATH_FLAG = FIX_RPATH_FLAG | DEPLOY_ONLY_LIBRARIES_FLAG
)

type DeployRequest struct {
  sourcePath string // relative or absolute path of file to process
  sourceRoot string // if empty then sourcePath is absolute path
  targetPath string // target *relative* path
  flags Bitmask // deployment flags
}

func (dp *DeployRequest) FullPath() string {
  if len(dp.sourceRoot) == 0 {
    return dp.sourcePath
  } else {
    return filepath.Join(dp.sourceRoot, dp.sourcePath)
  }
}

func (dp *DeployRequest) Basename() string {
  return filepath.Base(dp.sourcePath)
}

func (dp *DeployRequest) SourceDir() string {
  return filepath.Dir(dp.sourcePath)
}

func (dp *DeployRequest) RequiresRPathFix() bool {
  return dp.flags.HasFlag(FIX_RPATH_FLAG)
}

func (dp *DeployRequest) IsLddDependency() bool {
  return dp.flags.HasFlag(LDD_DEPENDENCY_FLAG)
}

type AppDeployer struct {
  waitGroup sync.WaitGroup
  processedLibs map[string]bool

  libsChannel chan *DeployRequest
  copyChannel chan *DeployRequest
  stripChannel chan string
  rpathChannel chan string
  qtChannel chan string

  qtDeployer *QtDeployer
  additionalLibPaths []string
  destinationPath string
  targetExePath string
}

func (ad *AppDeployer) DeployApp() {
  if err := ad.qtDeployer.queryQtEnv(); err != nil {
    log.Println(err)
  }

  ad.waitGroup.Add(1)
  go ad.processMainExe()

  go ad.processCopyTasks()
  go ad.processFixRPathTasks()
  go ad.processStripTasks()
  go ad.processQtLibTasks()

  blacklist := generateLibsBlacklist()

  log.Printf("Waiting for processing to finish")
  ad.waitGroup.Wait()
  log.Printf("Processing has finished")

  close(ad.libsChannel)
  close(ad.copyChannel)
  close(ad.qtChannel)
  close(ad.rpathChannel)
  close(ad.stripChannel)

  err := cleanupBlacklistedLibs(ad.LibsPath(), blacklist)
  if err != nil { log.Printf("Error while removing blacklisted libs: %v", err) }

  // let channels goroutines print end of work confirmation
  time.Sleep(200 * time.Millisecond)
}

func (ad *AppDeployer) LibsPath() string {
  return filepath.Join(ad.destinationPath, "lib")
}

func (ad *AppDeployer) addLibTask(sourceRoot, sourcePath, targetPath string, flags Bitmask) {
  ad.waitGroup.Add(1)
  go func() {
    ad.libsChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: sourcePath,
      targetPath: targetPath,
      flags: flags,
    }
  }()
}

func (ad *AppDeployer) addCopyTask(sourceRoot, sourcePath, targetPath string, flags Bitmask) {
  ad.waitGroup.Add(1)
  go func() {
    ad.copyChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: sourcePath,
      targetPath: targetPath,
      flags: flags,
    }
  }()
}

func (ad *AppDeployer) accountLibrary(libpath string) {
  log.Printf("Processed library %v", libpath)
  ad.processedLibs[libpath] = true
}

func (ad *AppDeployer) isLibraryDeployed(libpath string) bool {
  _, ok := ad.processedLibs[libpath]
  return ok
}

func (ad *AppDeployer) processMainExe() {
  dependencies, err := ad.findLddDependencies(filepath.Base(ad.targetExePath), ad.targetExePath)
  if err != nil { log.Fatal(err) }

  ad.accountLibrary(ad.targetExePath)
  ad.addCopyTask("", ad.targetExePath, ".", LDD_AND_RPATH_FLAG)

  for _, dependPath := range dependencies {
    if !ad.isLibraryDeployed(dependPath) {
      ad.addLibTask("", dependPath, "lib", LDD_AND_RPATH_FLAG)
    } else {
      log.Printf("Dependency seems to be processed: %v", dependPath)
    }
  }

  go ad.processLibTasks()

  ad.waitGroup.Done()
  log.Println("Main exe processing finished")
}

func (ad *AppDeployer) processCopyTasks() {
  copiedFiles := make(map[string]bool)

  for copyRequest := range ad.copyChannel {
    ad.processCopyTask(copiedFiles, copyRequest)
    ad.waitGroup.Done()
  }

  log.Printf("Copy tasks processing finished")
}

func (ad *AppDeployer) processCopyTask(copiedFiles map[string]bool, copyRequest *DeployRequest) {
  var destinationPath, destinationPrefix string

  if len(copyRequest.sourceRoot) == 0 {
    // absolute path
    destinationPrefix = copyRequest.targetPath
  } else {
    destinationPrefix = filepath.Join(copyRequest.targetPath, copyRequest.SourceDir())
  }

  sourcePath := copyRequest.FullPath()
  destinationPath = filepath.Join(ad.destinationPath, destinationPrefix, filepath.Base(copyRequest.sourcePath))

  if _, ok := copiedFiles[destinationPath]; ok {
    log.Printf("File %v has already been copied", sourcePath)
    return
  }

  ensureDirExists(destinationPath)
  err := copyFile(sourcePath, destinationPath)

  if err != nil {
    log.Printf("Error while copying [%v] to [%v]: %v", sourcePath, destinationPath, err)
    return
  }

  copiedFiles[destinationPath] = true
  log.Printf("Copied [%v] to [%v]", sourcePath, destinationPath)
  isQtLibrary := false

  if copyRequest.IsLddDependency() {
    libraryBasename := filepath.Base(destinationPath)
    libname := strings.ToLower(libraryBasename)

    if strings.HasPrefix(libname, "libqt") {
      ad.addQtLibTask(destinationPath)
      isQtLibrary = true
    }
  }

  if !isQtLibrary && copyRequest.RequiresRPathFix() {
    ad.addFixRPathTask(destinationPath)
  }
}

func (ad *AppDeployer) addFixRPathTask(fullpath string) {
  ad.waitGroup.Add(1)
  go func() {
    ad.rpathChannel <- fullpath
  }()
}

func (ad *AppDeployer) addQtLibTask(fullpath string) {
  if !ad.qtDeployer.qtEnvironmentSet {
    log.Println("Qt environment is not set!")
    return
  }

  ad.waitGroup.Add(1)
  go func() {
    ad.qtChannel <- fullpath
  }()
}

// copies everything without inspection
func (ad *AppDeployer) copyRecursively(sourceRoot, sourcePath, targetPath string) error {
  // rescue agains premature finish of the main loop
  ad.waitGroup.Add(1)
  defer ad.waitGroup.Done()

  var emptyFlags Bitmask = 0

  rootpath := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Copying recursively %v into %v", rootpath, targetPath)

  err := filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    relativePath, err := filepath.Rel(sourceRoot, path)
    if err != nil {
      log.Println(err)
    }

    ad.addCopyTask(sourceRoot, relativePath, targetPath, emptyFlags)

    return nil
  })

  return err
}

// inspects libraries for dependencies and copies other files
func (ad *AppDeployer) deployRecursively(sourceRoot, sourcePath, targetPath string, flags Bitmask) error {
  // rescue agains premature finish of the main loop
  ad.waitGroup.Add(1)
  defer ad.waitGroup.Done()

  rootpath := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Deploying recursively %v in %v to %v", sourcePath, sourceRoot, targetPath)

  onlyLibraries := flags.HasFlag(DEPLOY_ONLY_LIBRARIES_FLAG)
  var emptyFlags Bitmask = 0

  err := filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    basename := filepath.Base(path)
    isLibrary := strings.HasPrefix(basename, "lib") && strings.Contains(basename, ".so")

    if !isLibrary && onlyLibraries {
      return nil
    }

    relativePath, err := filepath.Rel(sourceRoot, path)
    if err != nil {
      log.Println(err)
    }

    if isLibrary {
      ad.addLibTask(sourceRoot, relativePath, targetPath, flags | LDD_DEPENDENCY_FLAG)
    } else {
      ad.addCopyTask(sourceRoot, relativePath, targetPath, emptyFlags)
    }

    return nil
  })

  return err
}
