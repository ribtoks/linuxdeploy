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
)

const (
  // constants for parameter of deployRecursively() method
  DEPLOY_EVERYTHING = false
  DEPLOY_LIBRARIES = true
  LDD_DEPENDENCY = true
  ORDINARY_FILE = false
)

type DeployRequest struct {
  sourcePath string // relative or absolute path of file to process
  sourceRoot string // if empty then sourcePath is absolute path
  targetPath string // target *relative* path
  isLddDependency bool // if true, check ldd dependencies
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
  go ad.processQtLibs()

  log.Printf("Waiting for processing to finish")
  ad.waitGroup.Wait()
  log.Printf("Processing has finished")

  close(ad.libsChannel)
  close(ad.copyChannel)
  close(ad.qtChannel)
  close(ad.rpathChannel)
}

func (ad *AppDeployer) deployLibrary(sourceRoot, sourcePath, targetPath string) {
  ad.waitGroup.Add(1)
  go func() {
    ad.libsChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: sourcePath,
      targetPath: targetPath,
      isLddDependency: true,
    }
  }()
}

func (ad *AppDeployer) copyFile(sourceRoot, sourcePath, targetPath string, isLddDependency bool) {
  ad.waitGroup.Add(1)
  go func() {
    ad.copyChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: sourcePath,
      targetPath: targetPath,
      isLddDependency: isLddDependency,
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
  dependencies, err := ad.findLddDependencies(ad.targetExePath)
  if err != nil { log.Fatal(err) }

  ad.accountLibrary(ad.targetExePath)
  ad.copyFile("", ad.targetExePath, ".", LDD_DEPENDENCY)

  for _, dependPath := range dependencies {
    if !ad.isLibraryDeployed(dependPath) {
      ad.deployLibrary("", dependPath, "lib")
    } else {
      log.Printf("Dependency seems to be processed: %v", dependPath)
    }
  }

  go ad.processLibs()

  ad.waitGroup.Done()
}

func (ad *AppDeployer) processCopyRequests() {
  for copyRequest := range ad.copyChannel {
    ad.processCopyRequest(copyRequest)
    ad.waitGroup.Done()
  }
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

  log.Printf("Copying %v to %v", sourcePath, destinationPath)
  err := copyFile(sourcePath, destinationPath)

  if err == nil && copyRequest.isLddDependency {
    ad.handleQtLibrary(destinationPath)
    ad.changeRPath(destinationPath)
  } else {
    log.Println(err)
  }

  // TODO: submit to strip/patchelf/etc. if copyRequest.isLddDependency
}

func (ad *AppDeployer) changeRPath(fullpath string) {
  ad.waitGroup.Add(1)
  go func() {
    ad.rpathChannel <- fullpath
  }()
}

func (ad *AppDeployer) handleQtLibrary(fullpath string) {
  ad.waitGroup.Add(1)
  go func() {
    ad.qtChannel <- fullpath
  }()
}

// copies one file
func (ad *AppDeployer) copyOnce(sourceRoot, sourcePath, targetPath string) error {
  path := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Copying once %v into %v", path, targetPath)
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
      isLddDependency: false,
    }
  }()

  return err
}

// copies everything without inspection
func (ad *AppDeployer) copyRecursively(sourceRoot, sourcePath, targetPath string) error {
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

    ad.copyFile(sourceRoot, relativePath, targetPath, ORDINARY_FILE)

    return nil
  })

  return err
}

// inspects libraries for dependencies and copies other files
func (ad *AppDeployer) deployRecursively(sourceRoot, sourcePath, targetPath string, onlyLibraries bool) error {
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
    isLibrary := strings.Contains(basename, ".so")

    if !isLibrary && onlyLibraries {
      return nil
    }

    relativePath, err := filepath.Rel(sourceRoot, path)
    if err != nil {
      log.Println(err)
    }

    if isLibrary {
      ad.deployLibrary(sourceRoot, relativePath, targetPath)
    } else {
      ad.copyFile(sourceRoot, relativePath, targetPath, ORDINARY_FILE)
    }

    return nil
  })

  return err
}
