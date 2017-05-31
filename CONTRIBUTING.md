## Basic architecture overview

The whole deployment process consists of several pipelines: libraries inspection, files copying, RPATH patching, binaries stripping, special Qt libraries handling and others. 
Each pipeline is represented by an appropriate `channel` which is being handled in it's own goroutine. Libraries and files are passed over from one pipeline to the other after being processed.

`AppDeployer` is a top-level entity to orchestrate the whole deployment. It kicks-off the process by calling `processMainExe()` and starting processign of all other pipelines like `processCopyTasks()`, `processStripTasks()` and others.

Another important place is deploying all Qt dependencies. `QtDeployer` as a part of `AppDeployer` is responsible for this. It handles plugins, qml imports and libraries separately in the `processQtLibTasks()` and `deployQmlImports()`.
Also `libQt5Core` needs to have hardcoded paths patched which is implemented in the `patchQtCore()` method. Qt environment is derived from the `qmake` output which is parsed in the beginning if Qt is in the dependencies or specified via `-qmake` param.

AppImage format is supported in a way of creating `AppRun` link, `.DirIcon` file and correct `.desktop` file (icon path without extension, Exec command and others). This is all handled in the `AppDeployer` respective methods which are called after copying the main exe file.

## How to contribute

- [Fork](http://help.github.com/forking/) linuxdeploy repository on GitHub
- Clone your fork locally
- Configure the upstream repo (`git remote add upstream git@github.com:Ribtoks/linuxdeploy.git`)
- Create local branch (`git checkout -b your_feature`)
- Work on your feature
- Build and Run tests (`go tests -v`)
- Push the branch to GitHub (`git push origin your_feature`)
- Send a [pull request](https://help.github.com/articles/using-pull-requests) on GitHub
