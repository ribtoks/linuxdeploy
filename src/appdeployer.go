package main

import (
  "log"
  "os"
  "os/exec"
  "strings"
  "sync"
  "path/filepath"
)

type DeployRequest struct {
  sourcePath string // relative or absolute path of file to process
  sourceRoot string // if empty then sourcePath is absolute path
  targetRoot string // target *relative* path
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

  libsChannel chan DeployRequest
  copyChannel chan DeployRequest
  stripChannel chan string
  rpathChannel chan string
  qtChannel chan DeployRequest

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
  go ad.processQtLibs()

  log.Printf("Waiting for processing to finish")
  ad.waitGroup.Wait()
  log.Printf("Processing has finished")
  close(ad.libsChannel)
  close(ad.copyChannel)
  close(ad.qtChannel)
}

func (ad *AppDeployer) processMainExe() {
  dependencies, err := ad.findLddDependencies(ad.targetExePath)
  if (err == nil) {
    ad.processedLibs[ad.targetExePath] = true

    ad.waitGroup.Add(1)
    go func() {
      ad.copyChannel <- DeployRequest{
        sourcePath: ad.targetExePath,
        targetRoot: ".",
        isLddDependency: true,
      }
    }()

    for _, dependPath := range dependencies {
      if _, ok := ad.processedLibs[dependPath]; !ok {
        ad.waitGroup.Add(1)
        go func(dlp string) {
          ad.libsChannel <- DeployRequest {
            sourcePath: dlp,
            targetRoot: "lib",
            isLddDependency: true,
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
    libpath := request.FullPath()
    log.Printf("Processing library: %v", libpath)

    if _, ok := ad.processedLibs[libpath]; !ok {
      dependencies, err := ad.findLddDependencies(libpath)
      if (err == nil) {
        ad.processedLibs[libpath] = true

        ad.waitGroup.Add(1)
        go func(copyRequest DeployRequest) {
          ad.copyChannel <- copyRequest
        }(request)

        for _, dependPath := range dependencies {
          if _, ok := ad.processedLibs[dependPath]; !ok {
            ad.waitGroup.Add(1)
            go func(dlp string, isLddDependency bool) {
              ad.libsChannel <- DeployRequest {
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

    ad.waitGroup.Done()
  }
}

func (ad *AppDeployer) processCopyRequests() {
  for copyRequest := range ad.copyChannel {

    var destinationPath, destinationPrefix string

    if len(copyRequest.sourceRoot) == 0 {
      // absolute path
      destinationPrefix = copyRequest.targetRoot
    } else {
      destinationPrefix = filepath.Join(copyRequest.targetRoot, copyRequest.SourceDir())
    }

    sourcePath := copyRequest.FullPath()
    destinationPath = filepath.Join(ad.destinationPath, destinationPrefix, filepath.Base(copyRequest.sourcePath))

    ensureDirExists(destinationPath)

    log.Printf("Copying %v to %v", sourcePath, destinationPath)
    err := copyFile(sourcePath, destinationPath)

    if err == nil && copyRequest.isLddDependency {
      ad.waitGroup.Add(1)
      go func(qtRequest DeployRequest) {
        ad.qtChannel <- qtRequest
      }(copyRequest)
    }

    // TODO: submit to strip/patchelf/etc. if copyRequest.isLddDependency

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

func (ad *AppDeployer) copyOnce(sourceRoot, sourcePath, targetRoot string) error {
  path := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Copying once %v into %v", path, targetRoot)
  relativePath, err := filepath.Rel(sourceRoot, path)
  if err != nil {
    log.Println(err)
  }

  go func() {
    ad.copyChannel <- DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: relativePath,
      targetRoot: targetRoot,
      isLddDependency: false,
    }
  }()

  return err
}

func (ad *AppDeployer) copyRecursively(sourceRoot, sourcePath, targetRoot string) error {
  rootpath := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Copying recursively %v into %v", rootpath, targetRoot)

  err := filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    go func() {
      relativePath, err := filepath.Rel(sourceRoot, path)
      if err != nil {
        log.Println(err)
      }

      ad.copyChannel <- DeployRequest{
        sourceRoot: sourceRoot,
        sourcePath: relativePath,
        targetRoot: targetRoot,
        isLddDependency: false,
      }
    }()

    return nil
  })

  return err
}

// designed to copy Qt plugins or other libraries
func (ad *AppDeployer) deployRecursively(sourceRoot, sourcePath, targetRoot string) error {
  rootpath := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Deploying recursively %v in %v", sourceRoot, sourcePath)

  err := filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    // copy only libraries
    basename := filepath.Base(path)
    isLibrary := strings.Contains(basename, ".so")

    relativePath, err := filepath.Rel(sourceRoot, path)
    if err != nil {
      log.Println(err)
    }

    request := DeployRequest {
      sourceRoot: sourceRoot,
      sourcePath: relativePath,
      targetRoot: targetRoot,
      isLddDependency: isLibrary,
    }

    if isLibrary {
      go func() {
        ad.libsChannel <- request
      }()
    } else {
      go func() {
        ad.copyChannel <- request
      }()
    }

    return nil
  })

  return err
}
