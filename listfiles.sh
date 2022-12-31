#!/bin/bash
set -eu
go list -f '{{ range .GoFiles }}{{ $.Dir }}/{{ . }} {{ end }}{{ range .CgoFiles }}{{ $.Dir }}/{{ . }} {{ end }}' ./... | sed 's,/,\\,g'
