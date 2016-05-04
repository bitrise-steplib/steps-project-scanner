#!/bin/bash

export THIS_SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

RESTORE='\033[0m'
RED='\033[00;31m'
YELLOW='\033[00;33m'
BLUE='\033[00;34m'
GREEN='\033[00;32m'

function color_echo {
	color=$1
	msg=$2
	echo -e "${color}${msg}${RESTORE}"
}

function echo_fail {
	msg=$1
	echo
	color_echo "${RED}" "${msg}"
	exit 1
}

function echo_warn {
	msg=$1
	color_echo "${YELLOW}" "${msg}"
}

function echo_info {
	msg=$1
	echo
	color_echo "${BLUE}" "${msg}"
}

function echo_details {
	msg=$1
	echo "  ${msg}"
}

function echo_done {
	msg=$1
	color_echo "${GREEN}" "  ${msg}"
}

function validate_required_input {
	key=$1
	value=$2
	if [ -z "${value}" ] ; then
		echo_fail "[!] Missing required input: ${key}"
	fi
}

#=======================================
# Main
#=======================================

#
# Validate parameters
echo_info "Configs:"
echo_details "* scan_dir: $scan_dir"
echo_details "* output_dir: $output_dir"
echo_details "* scan_result_submit_url: $scan_result_submit_url"
echo_details "* scan_result_submit_api_token: $scan_result_submit_api_token"

echo

validate_required_input "scan_dir" $scan_dir

#
# Create scanner bin
tmp_dir=$(mktemp -d)
current_dir=$(pwd)

export ARCH=x86_64
export GOARCH=amd64

current_os=$(uname -s)
if [[ "$current_os" == "Darwin" ]] ; then
  export OS=Darwin
  export GOOS=darwin
elif [[ "$current_os" == "Darwin" ]]; then
  export OS=Linux
  export GOOS=linux
else
  echo_fail "step runs on unsupported os: $current_os"
fi

bin_pth="$tmp_dir/scanner"
scanner_go_path="$THIS_SCRIPTDIR/go/src/github.com/bitrise-core/bitrise-init"

cd $scanner_go_path
go build -o "$bin_pth"
cd $current_dir

#
# Run scanner
$bin_pth config --dir $scan_dir --output-dir $output_dir
