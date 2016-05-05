#!/bin/bash

set -e

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

echo

validate_required_input "scan_dir" $scan_dir
validate_required_input "output_dir" $output_dir

#
# Create scanner bin
echo_info "Create scanner bin..."

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

echo_done "ceated at: ${bin_pth}"

#
# Running scanner
echo_info "Running scanner..."

$bin_pth config --dir $scan_dir --output-dir $output_dir

echo
echo_done "scan finished"

#
# Submitting results
if [ ! -z "${scan_result_submit_url}" ] ; then
	if [[ -z "${CI}" ]] ; then
		echo_warn "scan_result_submit_url defined but step runs in NOT CI mode"
		echo_fail "only run in CI mode generated result to upload"
	fi

	if [[ ! -f "${output_dir}/result.json" ]] ; then
		echo_warn "no scan result found at ${output_dir}/result.json"
		echo_fail "nothing to upload"
	fi

	echo_info "Submitting results..."

	curl --fail -H "Content-Type: application/json" \
		--data-binary @${output_dir}/result.json \
		"${scan_result_submit_url}"

	echo
	echo_done "submitted"
fi
