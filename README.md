# Depgo

A Go file server built on AList with an original React web interface. Linux-first: the server runs from the terminal and files are managed in the browser. Windows is a secondary compatibility target.

## Development status

The first release is under active development. The current frontend covers sign-in, file and folder operations, shares with editable URLs, and PDF/image previews. Trash, version history, the text editor, and the local-disk-only safe startup flow have not yet been verified end to end.

**For now, do not use it with data that is exposed to the internet or exists only in one copy.** The backend base still contains upstream AList features. Start with a temporary data directory and backed-up sample files.

## Repository layout

- `backend/`: AList Go/Gin/GORM core plus Depgo changes.
- `frontend/`: React, TypeScript, Vite, Tailwind interface.
- `temp/`: reference sources kept outside Git; not needed to build.

## Building from source on Linux

You need Go 1.25 or newer and a Node.js version supported by the pinned Vite release. Versions are defined in `backend/go.mod` and `frontend/package.json`.

From the project root:

```sh
npm --prefix frontend ci
npm --prefix frontend run build
mkdir -p build
go build -C backend -o ../build/depgo .
```

The frontend output is written to `backend/public/dist/` and embedded into the Go binary, so build the frontend before the Go binary.

Development run with the current upstream CLI:

```sh
./build/depgo server --data ./data --log-std
```

Local configuration is stored in the `--data` directory. This section will be updated once the initial admin and local disk setup flow is complete. During development, do not expose the server to external networks before reviewing the default listen address and enabled services.

To develop the frontend, use a separate terminal:

```sh
npm --prefix frontend run dev
```

Vite proxies API requests to the backend at `localhost:5244`.

## Checks

```sh
npm --prefix frontend test
npm --prefix frontend run typecheck
npm --prefix frontend run build
go -C backend test ./drivers/local ./server/handles ./server/common ./server/middlewares
```

The full upstream test tree has known blockers involving Windows FUSE headers, Go 1.26 vet checks, and network-timing-dependent tests. Passing targeted tests does not mean the whole tree passes. A Linux cross-compile is not a substitute for running the binary on Linux.

## Data safety and scope

The first release targets local disk only, a single administrator, and anonymous share viewers. Nextcloud client compatibility, Office editing, and a remote storage UI are out of scope. PDF viewing relies on the browser's built-in reader, with a download fallback when it is unsupported.

Version history is not a backup. The database, version area, and trash area must be backed up consistently together with live files. Changes made to files outside the application are not captured as versions.

## License and provenance

The backend is derived from [AList](https://github.com/alist-org/alist). The AGPL-3.0 text is in `LICENSE`; provenance and attribution are described in `NOTICE`. Upstream copyright notices are preserved.

The frontend user interface layout, dimensions, and interaction patterns are inspired by [Nextcloud Files](https://github.com/nextcloud/server/tree/master/apps/files) (Nextcloud GmbH and contributors, AGPL-3.0). Components are clean React implementations adhering to Depgo's standalone architecture.

Depgo sources: <https://github.com/thozoz/depgo>. If you distribute a modified version or serve it over a network, keep the corresponding source available under the AGPL terms.
