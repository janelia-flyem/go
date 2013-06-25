h2. janelia-flyem Go repo

This repo snapshots a collection of mirrored go packages.  It serves the following purposes:

- Provides versioning of Go package dependencies and prevents trunk changes from breaking dependent programs.
- Removes dependencies for non-git version control systems like mercurial and bazaar.  Instead we just require git.
- Simplifies the build process by reducing multiple 'go get' calls to one.
