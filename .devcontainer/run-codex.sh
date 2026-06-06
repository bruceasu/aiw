#!/usr/bin/env bash
# run-fast.sh
# ============================

# 获取APP所在的目录的绝对路径
function get_abs_dir() {
    SOURCE="${BASH_SOURCE[0]}"
    # resolve $SOURCE until the file is no longer a symlink
    while [ -h "$SOURCE" ]; do
        TARGET="$(readlink "$SOURCE")"
        if [[ ${SOURCE} == /* ]]; then
            # echo "SOURCE '$SOURCE' is an absolute symlink to '$TARGET'"
            SOURCE="$TARGET"
        else
            DIR="$(dirname "$SOURCE")"
            # echo "SOURCE '$SOURCE' is a relative symlink to '$TARGET' (relative to '$DIR')"
            # if $SOURCE was a relative symlink, we need to resolve it
            # relative to the path where the symlink file was located
            SOURCE="$DIR/$TARGET"
        fi
    done
    # echo "SOURCE is '$SOURCE'"

    # RDIR="$( dirname "$SOURCE" )"
    DIR="$( cd -P "$( dirname "$SOURCE" )" && cd .. && pwd )"
    #DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
    # if [ "$DIR" != "$RDIR" ]; then
    #     echo "DIR '$RDIR' resolves to '$DIR'"
    # fi
    # echo "DIR is '$DIR'"
    echo $DIR
}



unixformat(){
  #check dos2unix exist
  if ! which dos2unix &>/dev/null
  then
      sed -i 's/\r//' $1
  else
      dos2unix $1 &>/dev/null
  fi
}

DEFAULT_PROFILE="balanced"
if [ -z "$1" ]; then
  echo "No profile specified. Using default profile: $DEFAULT_PROFILE"
  echo "Usage: $0 [profile]"
  echo "Available profiles: fast, architect, balanced, heavy, etc."
  echo "Example: $0 fast"
else
  echo "Using profile: $1"
fi


case $1 in
	fast|architect|balanced|heavy)
        PROFILE="${1:-$DEFAULT_PROFILE}"
        shift;;
    *)
        echo "Using default profile: $DEFAULT_PROFILE"
        PROFILE="$DEFAULT_PROFILE";;
esac

CWD=`pwd`
SRC_DIR=`get_abs_dir`
cd $SRC_DIR

unixformat $SRC_DIR/.env

docker run -it --rm \
  --env-file $SRC_DIR/.env \
  --memory=4g \
  --cpus=2 \
  -v "$PWD:/workspace" \
  -v "$PWD/ai.sh:/usr/bin/ai.sh" \
  -v "$HOME/.m2:/root/.m2" \
  -v "$HOME/.codex:/root/.codex" \
  -v "$HOME/.gitconfig:/root/.gitconfig:ro" \
  -v "/mnt/libs:/mnt/libs" \
  -v "/mnt/temp:/mnt/temp" \
  -w /workspace \
  victor/java25 \
  ai.sh "$PROFILE" "$SRC_DIR/task.txt"
#  codex --profile "$PROFILE" "$@"
#  bash
cd $CWD
