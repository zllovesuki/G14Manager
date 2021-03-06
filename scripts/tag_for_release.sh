#!/bin/bash

changes=$(git log $1 --pretty=format:"(%h) - %s\n")
changelog="$2\n $changes"
echo -e $changelog
read -p "looks good?" yn
case $yn in
    [Yy]* ) echo -e $changelog | git tag -a -F- $2;;
    [Nn]* ) exit;;
    * ) echo "Please answer yes or no.";;
esac