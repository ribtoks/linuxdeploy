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

  libsChannel chan *DeployRequest
  copyChannel chan *DeployRequest
  stripChannel chan string
  rpathChannel chan string
  qtChannel chan *DeployRequest

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

func (ad *AppDeployer) processMainExe() {
  dependencies, err := ad.findLddDependencies(ad.targetExePath)
  if (err == nil) {
    ad.processedLibs[ad.targetExePath] = true

    ad.waitGroup.Add(1)
    go func() {
      ad.copyChannel <- &DeployRequest{
        sourcePath: ad.targetExePath,
        targetRoot: ".",
        isLddDependency: true,
      }
    }()

    for _, dependPath := range dependencies {
      if _, ok := ad.processedLibs[dependPath]; !ok {
        ad.waitGroup.Add(1)
        go func(dlp string) {
          ad.libsChannel <- &DeployRequest {
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
    go func(qtRequest *DeployRequest) {
      ad.qtChannel <- qtRequest
    }(copyRequest)

    ad.waitGroup.Add(1)
    go func(fullpath string) {
      ad.rpathChannel <- fullpath
    }(destinationPath)
  }

  // TODO: submit to strip/patchelf/etc. if copyRequest.isLddDependency
}

// copies one file
func (ad *AppDeployer) copyOnce(sourceRoot, sourcePath, targetRoot string) error {
  path := filepath.Join(sourceRoot, sourcePath)
  log.Printf("Copying once %v into %v", path, targetRoot)
  relativePath, err := filepath.Rel(sourceRoot, path)
  if err != nil {
    log.Println(err)
  }

  ad.waitGroup.Add(1)
  go func() {
    ad.copyChannel <- &DeployRequest{
      sourceRoot: sourceRoot,
      sourcePath: relativePath,
      targetRoot: targetRoot,
      isLddDependency: false,
    }
  }()

  return err
}

// copies everything without inspection
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

    relativePath, err := filepath.Rel(sourceRoot, path)
    if err != nil {
      log.Println(err)
    }

    ad.waitGroup.Add(1)
    go func() {
      ad.copyChannel <- &DeployRequest{
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

// inspects libraries for dependencies and copies other files
func (ad *AppDeployer) deployRecursively(sourceRoot, sourcePath, targetRoot string, onlyLibraries bool) error {
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

    request := &DeployRequest {
      sourceRoot: sourceRoot,
      sourcePath: relativePath,
      targetRoot: targetRoot,
      isLddDependency: isLibrary,
    }

    ad.waitGroup.Add(1)
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
