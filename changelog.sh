#!/bin/bash

changes=$(git log $1 --pretty=format:"[view commit](http://github.com/zllovesuki/ROGManager/commit/%H) %s")
changelog="# $2\n$changes"
echo -e "$changelog\n\n$(cat Changelog.md)" > Changelog.md