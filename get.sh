#!/bin/bash

# Repo-specific content: the repo is `vision`; the distributed binary and
# installed command is `vzn`. URLs use REPO, asset filenames and the install
# name use BIN.
export VERIFY_CHECKSUM=0
export OWNER=alexellis
export REPO=vision
export BIN=vzn
export BINLOCATION="/usr/local/bin"
export SUCCESS_CMD="$BINLOCATION/$BIN version"

###############################
# Content common across repos #
##################################

version=$(curl -sI https://github.com/$OWNER/$REPO/releases/latest | grep -i "location:" | awk -F"/" '{ printf "%s", $NF }' | tr -d '\r')
if [ ! $version ]; then
    echo "Failed while attempting to install $BIN. Please manually install:"
    echo ""
    echo "1. Open your web browser and go to https://github.com/$OWNER/$REPO/releases"
    echo "2. Download the latest release for your platform. Call it '$BIN'."
    echo "3. chmod +x ./$BIN"
    echo "4. mv ./$BIN $BINLOCATION"
    exit 1
fi

hasCli() {
    hasCurl=$(which curl)
    if [ "$?" = "1" ]; then
        echo "You need curl to use this script."
        exit 1
    fi
}

checkHash(){
    sha_cmd="sha256sum"
    if [ ! -x "$(command -v $sha_cmd)" ]; then
        sha_cmd="shasum -a 256"
    fi
    if [ -x "$(command -v $sha_cmd)" ]; then
        targetFileDir=${targetFile%/*}
        (cd $targetFileDir && curl -sSL $url.sha256 | $sha_cmd -c >/dev/null)
        if [ "$?" != "0" ]; then
            rm $targetFile
            echo "Binary checksum didn't match. Exiting"
            exit 1
        fi
    fi
}

getPackage() {
    uname=$(uname)
    userid=$(id -u)

    suffix=""
    case $uname in
    "Darwin")
        arch=$(uname -m)
        case $arch in
        "x86_64")
        suffix="-darwin"
        ;;
        esac
        case $arch in
        "arm64")
        suffix="-darwin-arm64"
        ;;
        esac
    ;;
    "MINGW"*)
        suffix=".exe"
        BINLOCATION="$HOME/bin"
        mkdir -p $BINLOCATION
    ;;
    "Linux")
        arch=$(uname -m)
        case $arch in
        "aarch64")
        suffix="-arm64"
        ;;
        esac
        case $arch in
        "armv6l" | "armv7l")
        suffix="-armhf"
        ;;
        esac
    ;;
    esac

    targetFile="/tmp/$BIN$suffix"

    if [ "$userid" != "0" ]; then
        targetFile="$(pwd)/$BIN$suffix"
    fi

    if [ -e "$targetFile" ]; then
        rm "$targetFile"
    fi

    url=https://github.com/$OWNER/$REPO/releases/download/$version/$BIN$suffix
    echo "Downloading package $url as $targetFile"

    curl -sSL $url --output "$targetFile"

    if [ "$?" = "0" ]; then
        if [ "$VERIFY_CHECKSUM" = "1" ]; then
            checkHash
        fi

        chmod +x "$targetFile"

        echo "Download complete."

        if [ ! -w "$BINLOCATION" ]; then
            echo
            echo "============================================================"
            echo "  The script was run as a user who is unable to write"
            echo "  to $BINLOCATION. To complete the installation the"
            echo "  following commands may need to be run manually."
            echo "============================================================"
            echo
            echo "  sudo cp $BIN$suffix $BINLOCATION/$BIN"
            echo
        else
            echo
            echo "Running with sufficient permissions to attempt to move $BIN to $BINLOCATION"

            if [ ! -w "$BINLOCATION/$BIN" ] && [ -f "$BINLOCATION/$BIN" ]; then
                echo
                echo "================================================================"
                echo "  $BINLOCATION/$BIN already exists and is not writeable"
                echo "  by the current user.  Please adjust the binary ownership"
                echo "  or run sh/bash with sudo."
                echo "================================================================"
                echo
                exit 1
            fi

            mv $targetFile $BINLOCATION/$BIN

            if [ "$?" = "0" ]; then
                echo "New version of $BIN installed to $BINLOCATION"
            fi

            if [ -e "$targetFile" ]; then
                rm "$targetFile"
            fi

            ${SUCCESS_CMD}
        fi
    fi
}

hasCli
getPackage
