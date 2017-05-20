package main

import (
  "log"
  "os"
  "os/exec"
  "strings"
  "sync"
  "path/filepath"
)

type CopyRequest struct {
  sourceRoot string // if empty then sourcePath is absolute path
  sourcePath string
  targetRoot string
  isAppDependency bool
}

type DependencyRequest struct {
  sourcePath string
  isAppDependency bool
}

type AppDeployer struct {
  waitGroup sync.WaitGroup
  processedLibs map[string]bool

  libsChannel chan *DependencyRequest
  copyChannel chan *CopyRequest
  stripChannel chan string
  rpathChannel chan string
  qtChannel chan string

  additionalLibPaths []string
  destinationPath string
  targetExePath string
}

func (ad *AppDeployer) DeployApp() {
  ad.waitGroup.Add(1)

  go ad.processMainExe()
  go ad.processCopyRequests()

  log.Printf("Waiting for processing to finish")
  ad.waitGroup.Wait()
  log.Printf("Processing has finished")
  close(ad.libsChannel)
  close(ad.copyChannel)
}

func (ad *AppDeployer) processMainExe() {
  dependencies, err := ad.findLddDependencies(ad.targetExePath)
  if (err == nil) {
    ad.processedLibs[ad.targetExePath] = true

    ad.waitGroup.Add(1)
    go func() {
      ad.copyChannel <- &CopyRequest{
        sourcePath: ad.targetExePath,
        targetRoot: ".",
        isAppDependency: true,
      }
    }()

    for _, dependPath := range dependencies {
      if _, ok := ad.processedLibs[dependPath]; !ok {
        ad.waitGroup.Add(1)
        go func(dlp string) {
          ad.libsChannel <- &DependencyRequest {
            sourcePath: dlp,
            isAppDependency: true,
          }
        }(dependPath)
      } else {
        log.Printf("Dependency seems to be processed: %v", dependPath)
      }
    }
  } else {
    log.Fatal(err)
  }

  go ad.processLibs()

  ad.waitGroup.Done()
}

func (ad *AppDeployer) processLibs() {
  for request := range ad.libsChannel {
    libpath := request.sourcePath

    if _, ok := ad.processedLibs[libpath]; !ok {
      dependencies, err := ad.findLddDependencies(libpath)
      if (err == nil) {
        ad.processedLibs[libpath] = true

        ad.waitGroup.Add(1)
        go func(libtocopy string, isAppDependency bool) {
          ad.copyChannel <- &CopyRequest{
            sourcePath: libtocopy,
            targetRoot: "lib",
            isAppDependency: isAppDependency,
          }
        }(libpath, request.isAppDependency)

        for _, dependPath := range dependencies {
          if _, ok := ad.processedLibs[dependPath]; !ok {
            ad.waitGroup.Add(1)
            go func(dlp string, isAppDependency bool) {
              ad.libsChannel <- &DependencyRequest {
                sourcePath: dlp,
                isAppDependency: isAppDependency,
              }
            }(dependPath, request.isAppDependency)
          }
        }
      } else {
        log.Printf("Error while dependency check: %v", err)
      }
    }

    ad.waitGroup.Done()
  }
}

func (ad *AppDeployer) processCopyRequests() {
  for copyRequest := range ad.copyChannel {

    var sourcePath, destinationPath, destinationPrefix string

    if len(copyRequest.sourceRoot) == 0 {
      // absolute path
      destinationPrefix = copyRequest.targetRoot
      sourcePath = copyRequest.sourcePath
    } else {
      destinationPrefix = filepath.Join(copyRequest.targetRoot, copyRequest.sourcePath)
      sourcePath = filepath.Join(copyRequest.sourceRoot, copyRequest.sourcePath)
    }

    destinationPath = filepath.Join(ad.destinationPath, destinationPrefix, filepath.Base(copyRequest.sourcePath))

    ensureDirExists(destinationPath)

    log.Printf("Copying %v to %v", sourcePath, destinationPath)
    copyFile(sourcePath, destinationPath)

    // TODO: submit to strip/patchelf/etc. if copyRequest.isAppDependency

    ad.waitGroup.Done()
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
