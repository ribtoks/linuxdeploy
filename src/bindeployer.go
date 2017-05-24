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
  "os/exec"
  "path/filepath"
  "strings"
  "os"
  "fmt"
)

func (ad *AppDeployer) processLibs() {
  for request := range ad.libsChannel {
    ad.processLibrary(request)
    ad.waitGroup.Done()
  }
}

func (ad *AppDeployer) processLibrary(request *DeployRequest) {
  libpath := request.FullPath()
  log.Printf("Processing library: %v", libpath)

  if ad.isLibraryDeployed(libpath) { return }

  dependencies, err := ad.findLddDependencies(libpath)
  if err != nil {
    log.Printf("Error while dependency check: %v", err)
    return
  }

  ad.accountLibrary(libpath)

  ad.waitGroup.Add(1)
  go func(copyRequest *DeployRequest) {
    ad.copyChannel <- copyRequest
  }(request)

  for _, dependPath := range dependencies {
    if !ad.isLibraryDeployed(dependPath) {
      ad.deployLibrary("", dependPath, "lib")
    }
  }
}

func (ad *AppDeployer) findLddDependencies(filepath string) ([]string, error) {
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

    log.Printf("Parsed lib %v from ldd [%v]", libpath, line)
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

func (ad *AppDeployer) processRunPathChangeRequests() {
  if _, err := exec.LookPath("patchelf"); err != nil {
    log.Printf("Patchelf cannot be found!")
    return
  }

  destinationRoot := ad.destinationPath

  for fullpath := range ad.rpathChannel {
    changeRPath(fullpath, destinationRoot)
    ad.waitGroup.Done()
  }
}

func changeRPath(fullpath, destinationRoot string) {
  libdir := filepath.Dir(fullpath)
  relativePath, err := filepath.Rel(destinationRoot, libdir)
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
