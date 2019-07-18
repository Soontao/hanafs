# Hana FS

Mount hana xs repository to local filesystem

## Installation

### Windows

1. Install [winfsp](https://github.com/billziss-gh/winfsp) library.
1. Download released binary file.

### MacOS

1. Install [osxfuse](https://osxfuse.github.io/) library.
1. Download released binary file.

### Linux

1. Just download released binary file.

## Features

* [x] Connect to hana repository, auth and fetch token
* [x] Read directory/file metadata
* [x] Cache directory/file metadata
* [x] Periodic refresh directory/file metadata
* [x] Read text/binary file content
* [x] Create files
* [x] Create directory
* [x] Correct timestamp & file size
* [x] Write data to file
* [ ] Editing locks
* [x] Move/Rename file
* [ ] Debug info
* [ ] Performance
* [ ] Upload binary files (images/...)
* [ ] Build executable binaries for windows/osx/linux
* [x] Refactor cache
* [ ] Deep load in startup
* [ ] CI
* [ ] Documentation & presentation

## Limitation

* File/directory status will be cached for better user experience, so that some properties will have some delay.
* Users' read/write operation without any cache, so that when user open/save file, OS/editor will be blocked. 
* You can **NOT** move file from one package to another package. 
* MacOS will not auto remove the mount point so that even you kill this application. So that the same name directory can be used as mount point one time before you restart.
* Unix `ln` and windows `shortcut` is not impl
* Please choose your own work package (instead of root package of hana) to improve the fs performance.

## [LICENSE](./LICENSE)
