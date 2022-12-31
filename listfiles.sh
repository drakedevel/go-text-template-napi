#!/bin/bash
set -eu
sed_fixup=''
dirsep='/'
if [[ "$OSTYPE" == "cygwin" || "$OSTYPE" == "msys" ]]; then
  sed_fixup='s,\\,\\\\,g'
  dirsep='\'
fi
go list -f "{{ range .GoFiles }}{{ \$.Dir }}${dirsep}{{ . }} {{ end }}{{ range .CgoFiles }}{{ \$.Dir }}${dirsep}{{ . }} {{ end }}" ./... | sed "$sed_fixup"
