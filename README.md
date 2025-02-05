[![CircleCI](https://circleci.com/gh/giantswarm/crd-docs-generator/tree/master.svg?style=shield&circle-token=c0f46d2b8c1482706d8d41b098d488efdf637a1f)](https://circleci.com/gh/giantswarm/crd-docs-generator/tree/master)
[![Docker Repository on Quay](https://quay.io/repository/giantswarm/crd-docs-generator/status "Docker Repository on Quay")](https://quay.io/repository/giantswarm/crd-docs-generator)

# crd-docs-generator

Generates schema reference documentation for Kubernetes Custom Resource Definitions (CRDs).

This tool is built to generate our [Management API CRD schema reference](https://docs.giantswarm.io/ui-api/management-api/crd/).

The generated output consists of Markdown files packed with HTML. By itself, this does not provide a fully readable and user-friendly set of documentation pages. Instead it relies on the HUGO website context, as the [giantswarm/docs](https://github.com/giantswarm/docs) repository, to provide an index page and useful styling.

## Assumptions/Prerequisites

This tool relies on:

- CRDs being defined in public source repositories
  - ... as one YAML file per CRD in the [apiextensions `config/crd` folder](https://github.com/giantswarm/apiextensions/tree/master/config/crd) folder
  - ... or in the [apiextensions `helm` folder](https://github.com/giantswarm/apiextensions/tree/master/helm) (deeper in the structure) as a file named `upstream.yaml`.
- Each CRD, identified by its full name, occurs only once
- CRDs providing an OpenAPIv3 validation schema
  - either in the `.spec.validation` section of a CRD containg only one version
  - or in the `.spec.versions[*].schema` position of a CRD containing multiple versions
- OpenAPIv3 schemas containing `description` attributes for every property.
- The topmost `description` value explaining the CRD itself. (For a CRD containing multiple versions, the first `description` found is used as such.)
- CR examples to be found in the [apiextensions `docs/cr` folder](https://github.com/giantswarm/apiextensions/tree/master/docs/cr) as one example per YAML file.

## Usage

The generator can be executed in Docker using a command like this:

```nohighlight
docker run \
    -v $PWD/path/to/output-folder:/opt/crd-docs-generator/output \
    -v $PWD:/opt/crd-docs-generator/config \
    quay.io/giantswarm/crd-docs-generator \
      --config /opt/crd-docs-generator/config/config.example.yaml
```

or in Go like this:

```nohighlight
go run main.go --config config.example.yaml
```

The volume mapping defines where the generated output will land.

## TODO

- Parse template only once instead of for every CRD
