#!/usr/bin/env bash
## Copyright (C) 2017, Oleksandr Kucherenko
## Last revisit: 2017-09-29 

## Script kindly provided via: https://gist.github.com/OleksandrKucherenko/9fb14f81a29b46886ccd63b774c5959f

# For help:
#   ./versionip.sh --help

# For developer / references:
#  https://ryanstutorials.net/bash-scripting-tutorial/bash-functions.php
#  http://tldp.org/LDP/abs/html/comparison-ops.html
#  https://misc.flogisoft.com/bash/tip_colors_and_formatting

## display help
function help(){
  echo 'usage: ./version-up.sh [-r|--release] [-a|--alpha] [-b|--beta] [-c|--release-candidate]'
  echo '                      [-m|--major] [-i|--minor] [-p|--patch] [-e|--revision] [-g|--git-revision]'
  echo '                      [--stay] [--default] [--help]'
  echo ''
  echo 'Switches:'
  echo '  --release           switch stage to release, no suffix, -r'
  echo '  --alpha             switch stage to alpha, -a'
  echo '  --beta              switch stage to beta, -b'
  echo '  --release-candidate switch stage to rc, -c'
  echo '  --major             increment MAJOR version part, -m'
  echo '  --minor             increment MINOR version part, -i'
  echo '  --patch             increment PATCH version part, -p'
  echo '  --revision          increment REVISION version part, -e'
  echo '  --git-revision      use git revision number as a revision part, -g'
  echo '  --stay              compose version.properties but do not do any increments, -s'
  echo '  --default           increment last found part of version, keeping the stage. Increment applied up to MINOR part.'
  echo '  --apply             run GIT command to apply version upgrade'
  echo ''
  echo 'Version: MAJOR.MINOR[.PATCH[.REVISION]][-STAGE]'
  echo ''
  echo 'Reference:'
  echo '  http://semver.org/'
  echo ''
  echo 'Versions priority:'
  echo '  1.0.0-alpha < 1.0.0-beta < 1.0.0-rc < 1.0.0'
  exit 0
}

## get highest version tag for all branches
function highest_tag(){
  local TAG=$(git tag --list 2>/dev/null | tail -n1 2>/dev/null)
  echo "$TAG"
}

## extract current branch name
function current_branch(){
  ## expected: heads/{branch_name}
  ## expected: {branch_name}
    local BRANCH=$(git rev-parse --abbrev-ref HEAD | cut -d"/" -f2)
    echo "$BRANCH"
}

## get latest/head commit hash number
function head_hash(){
  local COMMIT_HASH=$(git rev-parse --verify HEAD)
  echo "$COMMIT_HASH"
}

## extract tag commit hash code, tag name provided by argument
function tag_hash(){
  local TAG_HASH=$(git log -1 --format=format:"%H" $1 2>/dev/null | tail -n1)
  echo "$TAG_HASH"
}

## get latest tag in specified branch
function latest_tag(){
  #local TAG=$(git describe --tags $(git rev-list --tags --max-count=1 2>/dev/null) 2>/dev/null)
  local TAG=$(git describe --tags --abbrev=0 2>/dev/null)
  echo "$TAG"
}

## get latest revision number
function latest_revision(){
  local REV=$(git rev-list --count HEAD 2>/dev/null)
  echo "$REV"
}

## parse last found tag, extract it PARTS
function parse_last(){
  local position=$(($1-1))

  # two parts found only
  local SUBS=( ${PARTS[$position]//-/ } )
  #echo ${SUBS[@]}, size: ${#SUBS}

  # found NUMBER
  PARTS[$position]=${SUBS[0]}
  #echo ${PARTS[@]}

  # found SUFFIX
  if [[ ${#SUBS} -ge 1 ]]; then
    PARTS[4]=${SUBS[1],,} #lowercase
    #echo ${PARTS[@]}, ${SUBS[@]}
  fi
}

## increment REVISION part, don't touch STAGE
function increment_revision(){
  PARTS[3]=$(( PARTS[3] + 1 ))
  IS_DIRTY=1
}

## increment PATCH part, reset all other lower PARTS, don't touch STAGE
function increment_patch(){
  PARTS[2]=$(( PARTS[2] + 1 ))
  PARTS[3]=0
  IS_DIRTY=1
}

## increment MINOR part, reset all other lower PARTS, don't touch STAGE
function increment_minor(){
    PARTS[1]=$(( PARTS[1] + 1 ))
    PARTS[2]=0
    PARTS[3]=0
    IS_DIRTY=1
}

## increment MAJOR part, reset all other lower PARTS, don't touch STAGE
function incremet_major(){
  PARTS[0]=$(( PARTS[0] + 1 ))
  PARTS[1]=0
  PARTS[2]=0
  PARTS[3]=0
  IS_DIRTY=1
}

## increment the number only of last found PART: REVISION --> PATCH --> MINOR. don't touch STAGE
function increment_last_found(){
  if [[ "${#PARTS[3]}" == 0 || "${PARTS[3]}" == "0" ]]; then
    if [[ "${#PARTS[2]}" == 0 || "${PARTS[2]}" == "0" ]]; then
      increment_minor
    else
      increment_patch
    fi
  else
    increment_revision
  fi

  # stage part is not EMPTY
  if [[ "${#PARTS[4]}" != 0 ]]; then
    IS_SHIFT=1
  fi
}

## compose version from PARTS
function compose(){
  MAJOR="${PARTS[0]}"
  MINOR=".${PARTS[1]}"
  PATCH=".${PARTS[2]}"
  REVISION=".${PARTS[3]}"
  SUFFIX="-${PARTS[4]}"

  if [[ "${#PATCH}" == 1 ]]; then # if empty {PATCH}
    PATCH=""
  fi

  if [[ "${#REVISION}" == 1 ]]; then # if empty {REVISION}
    REVISION=""
  fi

  if [[ "${PARTS[3]}" == "0" ]]; then # if revision is ZERO
    REVISION=""
  fi

  # shrink patch and revision
  if [[ -z "${REVISION// }" ]]; then
    if [[ "${PARTS[2]}" == "0" ]]; then
      PATCH=""
    fi
  else # revision is not EMPTY
    if [[ "${#PATCH}" == 0 ]]; then
      PATCH=".0"
    fi
  fi

  # remove suffix if we don't have a alpha/beta/rc
  if [[ "${#SUFFIX}" == 1 ]]; then
    SUFFIX=""
  fi


  echo "${MAJOR}${MINOR}${PATCH}${REVISION}${SUFFIX}" #full format
}

# initial version used for repository without tags
INIT_VERSION=0.0.0.0-alpha

# do GIT data extracting
TAG=$(latest_tag)
REVISION=$(latest_revision)
BRANCH=$(current_branch)
TOP_TAG=$(highest_tag)
TAG_HASH=$(tag_hash $TAG)
HEAD_HASH=$(head_hash)

# do we have any GIT tag for parsing?!
echo ""
if [[ -z "${TAG// }" ]]; then
  TAG=$INIT_VERSION
  echo No tags found.
else
  echo Found tag: $TAG in branch \'$BRANCH\'
fi

# print current revision number based on number of commits
echo Current Revision: $REVISION
echo Current Branch: $BRANCH
echo ""

# if tag and branch commit hashes are different, than print info about that
#echo $HEAD_HASH vs $TAG_HASH
if [[ "$@" == "" ]]; then
    if [[ "$TAG_HASH" == "$HEAD_HASH" ]]; then
      echo "Tag $TAG and HEAD are aligned. We will stay on the TAG version."
      echo ""
      NO_ARGS_VALUE='--stay'
    else
      PATTERN="^[0-9]+.[0-9]+(.[0-9]+)*(-(alpha|beta|rc))*$"

      if [[ "$BRANCH" =~ $PATTERN ]]; then
        echo "Detected version branch '$BRANCH'. We will auto-increment the last version PART."
        echo ""
        NO_ARGS_VALUE='--default'
      else
        echo "Detected branch name '$BRANCH' than does not match version pattern. We will increase MINOR."
        echo ""
        NO_ARGS_VALUE='--minor'
      fi
    fi
fi

#
# {MAJOR}.{MINOR}[.{PATCH}[.{REVISION}][-(.*)]
#
#  Suffix: alpha, beta, rc
#    No Suffix --> {NEW_VERSION}-alpha
#    alpha --> beta
#    beta --> rc
#    rc --> {VERSION}
#
PARTS=( ${TAG//./ } )
parse_last ${#PARTS[@]} # array size as argument
#echo ${PARTS[@]}

# if no parameters than emulate --default parameter
if [[ "$@" == "" ]]; then
  set -- $NO_ARGS_VALUE
fi

# parse input parameters
for i in "$@"
do
  key="$i"

  case $key in
    -a|--alpha)                 # switched to ALPHA
    PARTS[4]="alpha"
    IS_SHIFT=1
    ;;
    -b|--beta)                  # switched to BETA
    PARTS[4]="beta"
    IS_SHIFT=1
    ;;
    -c|--release-candidate)     # switched to RC
    PARTS[4]="rc"
    IS_SHIFT=1
    ;;
    -r|--release)               # switched to RELEASE
    PARTS[4]=""
    IS_SHIFT=1
    ;;
    -p|--patch)                 # increment of PATCH
    increment_patch
    ;;
    -e|--revision)              # increment of REVISION
    increment_revision
    ;;
    -g|--git-revision)          # use git revision number as a revision part§
    PARTS[3]=$(( REVISION ))
    IS_DIRTY=1
    ;;
    -i|--minor)                 # increment of MINOR by default
    increment_minor
    ;;
    --default)                 # stay on the same stage, but increment only last found PART of version code
    increment_last_found
    ;;
    -m|--major)                 # increment of MAJOR
    incremet_major
    ;;
    -s|--stay)                  # extract version info
    IS_DIRTY=1
    NO_APPLY_MSG=1
    ;;
    --apply)
    DO_APPLY=1
    ;;
    -h|--help)
    help
    ;;
  esac
  shift
done

# detected shift, but no increment
if [[ "$IS_SHIFT" == "1" ]]; then
    # temporary disable stage shift
    stage=${PARTS[4]}
    PARTS[4]=''

    # detect first run on repository, INIT_VERSION was used
    if [[ "$(compose)" == "0.0" ]]; then
        increment_minor
    fi

    PARTS[4]=$stage
fi

# no increment applied yet and no shift of state, do minor increase
if [[ "$IS_DIRTY$IS_SHIFT" == "" ]]; then
    increment_minor
fi

# instruct user how to apply new TAG
echo -e "Proposed TAG: \033[32m$(compose)\033[0m"
echo ''

# is proposed tag in conflict with any other TAG
PROPOSED_HASH=$(tag_hash $(compose))
if [[ "${#PROPOSED_HASH}" -gt 0 && "$NO_APPLY_MSG" == "" ]]; then
  echo -e "\033[31mERROR:\033[0m "
  echo -e "\033[31mERROR:\033[0m Found conflict with existing tag \033[32m$(compose)\033[0m / $PROPOSED_HASH"
  echo -e "\033[31mERROR:\033[0m Only manual resolving is possible now."
  echo -e "\033[31mERROR:\033[0m "
  echo -e "\033[31mERROR:\033[0m To Resolve try to add --revision or --patch modifier."
  echo -e "\033[31mERROR:\033[0m "
  echo ""
fi

if [[ "$NO_APPLY_MSG" == "" ]]; then
    echo 'To apply changes manually execute the command(s):'
    echo -e "\033[90m"
    echo "  git tag $(compose)"
    echo "  git push origin $(compose)"
    echo -e "\033[0m"
fi

# compose version override file
if [[ "$TAG" == "$INIT_VERSION" ]]; then
    TAG='0.0'
fi
VERSION_FILE=version.properties

echo "# $(date)" >${VERSION_FILE}
echo snapshot.version=$(compose) >>${VERSION_FILE}
echo snapshot.lasttag=$TAG >>${VERSION_FILE}
echo snapshot.revision=$REVISION >>${VERSION_FILE}
echo snapshot.hightag=$TOP_TAG >>${VERSION_FILE}
echo snapshot.branch=$BRANCH >>${VERSION_FILE}
echo '# end of file' >>${VERSION_FILE}

# should we apply the changes
if [[ "$DO_APPLY" == "1" ]]; then
    echo ''
    echo "Applying git repository version up... no push, only local tag assignment!"
    echo ''

    git tag $(compose)

    # confirm that tag applied
    git --no-pager log --pretty=format:"%h%x09%Cblue%cr%Cgreen%x09%an%Creset%x09%s%Cred%d%Creset" -n 2 --date=short | nl -w2 -s"  "
fi

#
# Major logic of the script - "on each run script propose future version of the product".
#
#  - if no tags on project --> propose '0.1-alpha'
#  - do multiple build iterations until you become satisfied with result
#  - run 'version-up.sh --apply' to save result in GIT
#
