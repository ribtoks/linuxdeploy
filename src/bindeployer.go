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
  "os/exec"
  "path/filepath"
  "strings"
  "os"
  "fmt"
)

func (ad *AppDeployer) processLibTasks() {
  if _, err := exec.LookPath("ldd"); err != nil {
    log.Fatal("ldd cannot be found!")
  }

  for request := range ad.libsChannel {
    ad.processLibTask(request)
    ad.waitGroup.Done()
  }

  log.Println("Libraries processing finished")
}

func (ad *AppDeployer) processLibTask(request *DeployRequest) {
  libpath := request.FullPath()

  if ad.canSkipLibrary(libpath) {
    log.Printf("Skipping library: %v", libpath)
    return
  }

  log.Printf("Processing library: %v", libpath)

  dependencies, err := ad.findLddDependencies(request.Basename(), libpath)
  if err != nil {
    log.Printf("Error while dependency check for %v: %v", libpath, err)
    return
  }

  ad.accountLibrary(libpath)

  ad.waitGroup.Add(1)
  go func(copyRequest *DeployRequest) {
    ad.copyChannel <- copyRequest
  }(request)

  flags := request.flags
  // fix rpath of all the libs
  //flags.ClearFlag(FIX_RPATH_FLAG)
  flags.AddFlag(LDD_DEPENDENCY_FLAG)

  for _, dependPath := range dependencies {
    if !ad.isLibraryDeployed(dependPath) {
      ad.addLibTask("", dependPath, "lib", flags)
    }
  }
}

func (ad *AppDeployer) canSkipLibrary(libpath string) bool {
  canSkip := false
  if strings.HasPrefix(libpath, "linux-vdso.so") {
    canSkip = true
  } else if ad.isLibraryDeployed(libpath) {
    canSkip = true
  }

  return canSkip
}

func (ad *AppDeployer) findLddDependencies(basename, filepath string) ([]string, error) {
  log.Printf("Inspecting %v", filepath)

  out, err := exec.Command("ldd", filepath).Output()
  if err != nil { return nil, err }

  dependencies := make([]string, 0, 10)

  output := string(out)
  lines := strings.Split(output, "\n")
  for _, line := range lines {
    line = strings.TrimSpace(line)
    libname, libpath, err := parseLddOutputLine(line)

    if err != nil {
      log.Printf("Cannot parse ldd line: %v", line)
      continue
    }

    if len(libpath) == 0 {
      libpath = ad.resolveLibrary(libname)
    }

    log.Printf("[%v]: depends on %v from ldd [%v]", basename, libpath, line)
    dependencies = append(dependencies, libpath)
  }

  return dependencies, nil
}

func (ad *AppDeployer) addAdditionalLibPath(libpath string) {
  log.Printf("Adding addition libpath: %v", libpath)
  foundPath := libpath
  var err error

  if !filepath.IsAbs(foundPath) {
    if foundPath, err = filepath.Abs(foundPath); err == nil {
      log.Printf("Trying to resolve libpath to: %v", foundPath)

      if _, err = os.Stat(foundPath); os.IsNotExist(err) {
        exeDir := filepath.Dir(ad.targetExePath)
        foundPath = filepath.Join(exeDir, libpath)
        log.Printf("Trying to resolve libpath to: %v", foundPath)
      }
    }
  }

  if _, err := os.Stat(foundPath); os.IsNotExist(err) {
    log.Printf("Cannot find library path: %v", foundPath)
    return
  }

  log.Printf("Resolved additional libpath to: %v", foundPath)
  ad.additionalLibPaths = append(ad.additionalLibPaths, foundPath)
}

func (ad *AppDeployer) resolveLibrary(libname string) (foundPath string) {
  foundPath = libname

  for _, extraLibPath := range ad.additionalLibPaths {
    possiblePath := filepath.Join(extraLibPath, libname)

    if _, err := os.Stat(possiblePath); err == nil {
      foundPath = possiblePath
      break
    }
  }

  log.Printf("Resolving library %v to %v", libname, foundPath)
  return foundPath
}

func (ad *AppDeployer) processFixRPathTasks() {
  patchelfAvailable := true

  if _, err := exec.LookPath("patchelf"); err != nil {
    log.Printf("Patchelf cannot be found!")
    patchelfAvailable = false
  }

  destinationRoot := ad.destinationPath
  fixedFiles := make(map[string]bool)

  for fullpath := range ad.rpathChannel {
    if patchelfAvailable {
      if _, ok := fixedFiles[fullpath]; !ok {
        fixRPath(fullpath, destinationRoot)
        fixedFiles[fullpath] = true
      } else {
        log.Printf("RPATH has been already fixed for %v", fullpath)
      }
    }

    ad.addStripTask(fullpath)

    ad.waitGroup.Done()
  }

  log.Printf("RPath change requests processing finished")
}

func fixRPath(fullpath, destinationRoot string) {
  libdir := filepath.Dir(fullpath)
  relativePath, err := filepath.Rel(libdir, destinationRoot)
  if err != nil {
    log.Println(err)
    return
  }

  rpath := fmt.Sprintf("$ORIGIN:$ORIGIN/%s/lib/", relativePath)
  log.Printf("Changing RPATH for %v to %v", fullpath, rpath)

  cmd := exec.Command("patchelf", "--set-rpath", rpath, fullpath)
  if err = cmd.Run(); err != nil {
    log.Println(err)
  }
}

func (ad *AppDeployer) addStripTask(fullpath string) {
  if *stripFlag {
    ad.waitGroup.Add(1)
    go func() {
      ad.stripChannel <- fullpath
    }()
  }
}

func (ad *AppDeployer) processStripTasks() {
  stripAvailable := true

  if _, err := exec.LookPath("strip"); err != nil {
    log.Printf("Strip cannot be found!")
    stripAvailable = false
  }

  strippedBinaries := make(map[string]bool)

  for fullpath := range ad.stripChannel {
    if stripAvailable {
      if _, ok := strippedBinaries[fullpath]; !ok {
        stripBinary(fullpath)
      } else {
        log.Printf("%v has been already stripped", fullpath)
      }
    }

    ad.waitGroup.Done()
  }

  log.Printf("Strip requests processing finished")
}

func stripBinary(fullpath string) {
  log.Printf("Running strip on %v", fullpath)

  cmd := exec.Command("strip", fullpath)
  if err := cmd.Run(); err != nil {
    log.Println(err)
  }
}
