---
weight: 2
title: "Installation"
description: How to set up tfautomv on your workstation.
---

# Installation

Follow the instructions in any of the tabs below:

{{< tabs "installation" >}}
{{< tab "Homebrew" >}}

```bash
brew install padok-team/tap/tfautomv
```

{{< /tab >}}
{{< tab "Pre-compiled binary" >}}

On the Github reposotory's [Releases page](https://github.com/padok-team/tfautomv/releases),
download the binary that matches your workstation's OS and CPU architecture.

Put the binary in a directory present in your system's `PATH` environment
variable.

{{< /tab >}}
{{< tab "From source" >}}

You must have Go 1.18+ installed to compile tfautomv.

Clone the repository and build the binary:

```bash
git clone https://github.com/padok-team/tfautomv
cd tfautomv
make build
```

Then, move `bin/tfautomv` to a directory resent in your system's `PATH`
environment variable.

{{< /tab >}}
{{< /tabs >}}

Confirm that tfautomv is properly installed:

```bash
tfautomv -version
```

## Terraform version compatibility

`tfautomv` uses your local Terraform installation. The following version are supported:

- from `0.12.x` to `1.0.x`, only with the `-output=commands` option, since _moved blocks_ are not supported in these versions
- all Terraform version above `1.1.0`

## Next steps

Follow the [guided tutorial]({{< relref "getting-started/tutorial.md" >}}) to become familiar with tfautomv.
