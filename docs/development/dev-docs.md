# Documentation

## Local dev setup

You can run the documentation via:

```sh
make startdocs
```

If port `5173` is already in use, override the host port:

```sh
make startdocs DOCS_PORT=5174
```

You can remove the docs container image by running:

```sh
make cleandocs
```
