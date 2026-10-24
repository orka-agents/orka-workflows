# Textkit

A small POSIX shell library for turning text into slugs.

From the repository root:

```sh
. ./demo/textkit.sh
slug=$(slugify "Hello, Orka!")
printf '%s\n' "$slug" # hello-orka
```

Run the tests:

```sh
sh demo/test.sh
```
