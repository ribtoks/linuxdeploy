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

  if _, ok := ad.processedLibs[libpath]; !ok {
    dependencies, err := ad.findLddDependencies(libpath)
    if (err == nil) {
      ad.processedLibs[libpath] = true

      ad.waitGroup.Add(1)
      go func(copyRequest *DeployRequest) {
        ad.copyChannel <- copyRequest
      }(request)

      for _, dependPath := range dependencies {
        if _, ok := ad.processedLibs[dependPath]; !ok {
          ad.waitGroup.Add(1)
          go func(dlp string, isLddDependency bool) {
            ad.libsChannel <- &DeployRequest{
              sourcePath: dlp,
              targetRoot: "lib",
              isLddDependency: isLddDependency,
            }
          }(dependPath, request.isLddDependency)
        }
      }
    } else {
      log.Printf("Error while dependency check: %v", err)
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

    if err == nil {
      if len(libpath) == 0 {
        libpath = ad.resolveLibrary(libname)
      }

      log.Printf("Extracted %v from ldd [%v]", libpath, line)
      dependencies = append(dependencies, libpath)
    } else {
      log.Printf("Cannot parse ldd line: %v", line)
    }
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
  relativePath, err := filepath.Rel(destinationRoot, fullpath)
  if err != nil {
    log.Println(err)
    return
  }

  rpath := fmt.Sprintf("$ORIGIN:$ORIGIN/%s", relativePath)
  log.Printf("Changing RPATH for %v to %v", fullpath, rpath)

  cmd := exec.Command("patchelf", "--set-rpath", rpath, fullpath)
  if err = cmd.Run(); err != nil {
    log.Println(err)
  }
}
