h2. janelia-flyem Go repo

This repo snapshots a collection of mirrored go packages.  It serves two purposes:

- It provides versioning of Go package dependencies and prevents trunk changes from breaking dependent programs.
- It removes dependencies for non-git version control systems like mercurial and bazaar.  Instead we just require git.
