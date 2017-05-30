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
  // constants for parameter of deployRecursively() method
  DEPLOY_EVERYTHING = false
  DEPLOY_LIBRARIES = true
  LDD_DEPENDENCY = true
  ORDINARY_FILE = false
  FIX_RPATH = true
  LEAVE_RPATH = false
)

type DeployRequest struct {
  sourcePath string // relative or absolute path of file to process
  sourceRoot string // if empty then sourcePath is absolute path
  targetPath string // target *relative* path
  isLddDependency bool // if true, check ldd dependencies
  requiresRunPathFix bool // true for all qt plugins
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

  go ad.processCopyRequests()
  go ad.processRunPathChangeRequests()
  go ad.processStripRequests()
  go ad.processQtLibs()

  log.Printf("Waiting for processing to finish")
  ad.waitGroup.Wait()
  log.Printf("Processing has finished")

  close(ad.libsChannel)
  close(ad.copyChannel)
  close(ad.qtChannel)
  close(ad.rpathChannel)
  close(ad.stripChannel)
  
  // let channels goroutines print end of work confirmation
  time.Sleep(500 * time.Millisecond)
}

func (ad *AppDeployer) deployLibraryEx(sourceRoot, sourcePath, targetPath string, requiresRunPathFix bool) {
  ad.waitGroup.Add(1)
  go func() {
    ad.libsChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: sourcePath,
      targetPath: targetPath,
      isLddDependency: true,
      requiresRunPathFix: requiresRunPathFix,
    }
  }()
}

func (ad *AppDeployer) deployLibrary(sourceRoot, sourcePath, targetPath string) {
  ad.deployLibraryEx(sourceRoot, sourcePath, targetPath, false)
}

func (ad *AppDeployer) copyFileEx(sourceRoot, sourcePath, targetPath string, isLddDependency, requiresRunPathFix bool) {
  ad.waitGroup.Add(1)
  go func() {
    ad.copyChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: sourcePath,
      targetPath: targetPath,
      isLddDependency: isLddDependency,
      requiresRunPathFix: requiresRunPathFix,
    }
  }()
}

func (ad *AppDeployer) copyFile(sourceRoot, sourcePath, targetPath string, isLddDependency bool) {
  ad.copyFileEx(sourceRoot, sourcePath, targetPath, isLddDependency, false)
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
  dependencies, err := ad.findLddDependencies(ad.targetExePath)
  if err != nil { log.Fatal(err) }

  ad.accountLibrary(ad.targetExePath)
  ad.copyFileEx("", ad.targetExePath, ".", LDD_DEPENDENCY, FIX_RPATH)

  for _, dependPath := range dependencies {
    if !ad.isLibraryDeployed(dependPath) {
      ad.deployLibrary("", dependPath, "lib")
    } else {
      log.Printf("Dependency seems to be processed: %v", dependPath)
    }
  }

  go ad.processLibs()

  ad.waitGroup.Done()  
  log.Println("Main exe processing finished")
}

func (ad *AppDeployer) processCopyRequests() {
  for copyRequest := range ad.copyChannel {
    ad.processCopyRequest(copyRequest)
    ad.waitGroup.Done()
  }

  log.Printf("Copy requests processing finished")
}

func (ad *AppDeployer) processCopyRequest(copyRequest *DeployRequest) {
  var destinationPath, destinationPrefix string

  if len(copyRequest.sourceRoot) == 0 {
    // absolute path
    destinationPrefix = copyRequest.targetPath
  } else {
    destinationPrefix = filepath.Join(copyRequest.targetPath, copyRequest.SourceDir())
  }

  sourcePath := copyRequest.FullPath()
  destinationPath = filepath.Join(ad.destinationPath, destinationPrefix, filepath.Base(copyRequest.sourcePath))

  ensureDirExists(destinationPath)
  err := copyFile(sourcePath, destinationPath)

  if err != nil {
    log.Printf("Error while copying [%v] to [%v]: %v", sourcePath, destinationPath, err)
    return
  }
  
  log.Printf("Copied [%v] to [%v]", sourcePath, destinationPath)
  passedOver := false
  
  if copyRequest.isLddDependency {
    libraryBasename := filepath.Base(destinationPath)
    libname := strings.ToLower(libraryBasename)

    if strings.HasPrefix(libname, "libqt") {
      ad.handleQtLibrary(destinationPath)
      passedOver = true
    }
  }
  
  if !passedOver && copyRequest.requiresRunPathFix {
    ad.changeRPath(destinationPath)
  }
}

func (ad *AppDeployer) changeRPath(fullpath string) {
  ad.waitGroup.Add(1)
  go func() {
    ad.rpathChannel <- fullpath
  }()
}

func (ad *AppDeployer) handleQtLibrary(fullpath string) {
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

    ad.copyFileEx(sourceRoot, relativePath, targetPath, ORDINARY_FILE, LEAVE_RPATH)

    return nil
  })

  return err
}

// inspects libraries for dependencies and copies other files
func (ad *AppDeployer) deployRecursively(sourceRoot, sourcePath, targetPath string, onlyLibraries, fixRPath bool) error {
  // rescue agains premature finish of the main loop
  ad.waitGroup.Add(1)
  defer ad.waitGroup.Done()
  
  rootpath := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Deploying recursively %v in %v", sourceRoot, sourcePath)

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
      ad.deployLibraryEx(sourceRoot, relativePath, targetPath, fixRPath)
    } else {
      ad.copyFileEx(sourceRoot, relativePath, targetPath, ORDINARY_FILE, LEAVE_RPATH)
    }

    return nil
  })

  return err
}
